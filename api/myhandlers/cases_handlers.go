package myhandlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/authstore"
	"github.com/codemarked/go-lab/api/middleware"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/codemarked/go-lab/api/respond"
	"github.com/gin-gonic/gin"
)

func caseRowToJSON(r authstore.OperatorCaseRow) gin.H {
	out := gin.H{
		"id":                       r.ID,
		"created_at":               r.CreatedAt,
		"updated_at":               r.UpdatedAt,
		"status":                   r.Status,
		"priority":                 r.Priority,
		"subject_platform_user_id": r.SubjectPlatformUserID,
		"title":                    r.Title,
		"created_by_user_id":       r.CreatedByUserID,
	}
	if r.SubjectCharacterRef.Valid {
		out["subject_character_ref"] = r.SubjectCharacterRef.String
	}
	if r.Description.Valid {
		out["description"] = r.Description.String
	}
	if r.AssignedToUserID.Valid {
		out["assigned_to_user_id"] = int(r.AssignedToUserID.Int64)
	}
	return out
}

// ListOperatorCases GET /api/v1/cases
func ListOperatorCases(c *gin.Context) {
	limit := 50
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "limit must be between 1 and 200", nil)
			return
		}
		limit = n
	}
	status := strings.TrimSpace(c.Query("status"))
	var subjectUID *int
	if v := strings.TrimSpace(c.Query("subject_platform_user_id")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "subject_platform_user_id must be positive", nil)
			return
		}
		subjectUID = &n
	}
	var beforeID *int64
	if v := strings.TrimSpace(c.Query("before_id")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "before_id must be a positive integer", nil)
			return
		}
		beforeID = &n
	}
	rows, err := AuthStore.ListOperatorCases(c.Request.Context(), authstore.ListOperatorCasesQuery{
		Limit:         limit,
		Status:        status,
		SubjectUserID: subjectUID,
		BeforeID:      beforeID,
	})
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to list cases", nil)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		items = append(items, caseRowToJSON(r))
	}
	respond.OK(c, gin.H{"items": items, "limit": limit})
}

type createCaseBody struct {
	SubjectPlatformUserID int     `json:"subject_platform_user_id" binding:"required"`
	SubjectCharacterRef   *string `json:"subject_character_ref"`
	Title                 string  `json:"title" binding:"required,min=1,max=256"`
	Description           string  `json:"description"`
	Priority              string  `json:"priority"`
}

// PostOperatorCase POST /api/v1/cases
func PostOperatorCase(c *gin.Context) {
	uid, ok := middleware.AuthUserIDFromContext(c)
	if !ok {
		respond.Error(c, http.StatusForbidden, api.CodeForbidden, "user subject required", nil)
		return
	}
	var body createCaseBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid JSON body", nil)
		return
	}
	priority := strings.TrimSpace(body.Priority)
	if priority == "" {
		priority = authstore.CasePriorityNormal
	}
	var subjectExists int
	err := Database.Db.QueryRowContext(c.Request.Context(),
		`SELECT id FROM users WHERE id = ? AND deleted_at IS NULL`, body.SubjectPlatformUserID).Scan(&subjectExists)
	if errors.Is(err, sql.ErrNoRows) {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "subject_platform_user_id does not exist in users table", map[string]any{
			"field": "subject_platform_user_id",
			"id":    body.SubjectPlatformUserID,
		})
		return
	}
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to validate subject user", nil)
		return
	}
	id, err := AuthStore.CreateOperatorCase(c.Request.Context(), body.SubjectPlatformUserID, body.SubjectCharacterRef,
		strings.TrimSpace(body.Title), strings.TrimSpace(body.Description), priority, uid)
	if err != nil {
		if errors.Is(err, authstore.ErrInvalidCasePriority) {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, err.Error(), map[string]any{"field": "priority"})
			return
		}
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to create case", nil)
		return
	}
	reason := reasonFromContext(c)
	sub := c.GetString("auth_subject")
	rid := requestid.FromContext(c)
	_ = AuthStore.InsertAdminAuditEvent(c.Request.Context(), &uid, sub, "case.created", "operator_case", strconv.FormatInt(id, 10),
		reason, rid, c.ClientIP(), c.GetHeader("User-Agent"), nil)
	respond.JSONOK(c, http.StatusCreated, gin.H{"id": id, "status": authstore.CaseStatusOpen})
}

// GetOperatorCase GET /api/v1/cases/:id
func GetOperatorCase(c *gin.Context) {
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	row, err := AuthStore.GetOperatorCase(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "case not found", nil)
			return
		}
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to load case", nil)
		return
	}
	respond.OK(c, caseRowToJSON(row))
}

type patchCaseBody struct {
	Status        *string `json:"status"`
	Priority      *string `json:"priority"`
	AssignedToUID *int    `json:"assigned_to_user_id"`
}

// PatchOperatorCase PATCH /api/v1/cases/:id
func PatchOperatorCase(c *gin.Context) {
	uid, ok := middleware.AuthUserIDFromContext(c)
	if !ok {
		respond.Error(c, http.StatusForbidden, api.CodeForbidden, "user subject required", nil)
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var body patchCaseBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid JSON body", nil)
		return
	}
	err := AuthStore.UpdateOperatorCase(c.Request.Context(), id, body.Status, body.Priority, body.AssignedToUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "case not found", nil)
			return
		}
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, err.Error(), nil)
		return
	}
	reason := reasonFromContext(c)
	sub := c.GetString("auth_subject")
	rid := requestid.FromContext(c)
	meta, _ := json.Marshal(map[string]any{"case_id": id})
	_ = AuthStore.InsertAdminAuditEvent(c.Request.Context(), &uid, sub, "case.updated", "operator_case", strconv.FormatInt(id, 10),
		reason, rid, c.ClientIP(), c.GetHeader("User-Agent"), meta)
	row, _ := AuthStore.GetOperatorCase(c.Request.Context(), id)
	respond.OK(c, caseRowToJSON(row))
}

type postNoteBody struct {
	Body string `json:"body" binding:"required,min=1"`
}

// PostOperatorCaseNote POST /api/v1/cases/:id/notes
func PostOperatorCaseNote(c *gin.Context) {
	uid, ok := middleware.AuthUserIDFromContext(c)
	if !ok {
		respond.Error(c, http.StatusForbidden, api.CodeForbidden, "user subject required", nil)
		return
	}
	caseID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var body postNoteBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid JSON body", nil)
		return
	}
	if _, err := AuthStore.GetOperatorCase(c.Request.Context(), caseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "case not found", nil)
			return
		}
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to load case", nil)
		return
	}
	noteID, err := AuthStore.InsertOperatorCaseNote(c.Request.Context(), caseID, body.Body, uid)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to add note", nil)
		return
	}
	reason := reasonFromContext(c)
	sub := c.GetString("auth_subject")
	rid := requestid.FromContext(c)
	meta, _ := json.Marshal(map[string]any{"note_id": noteID})
	_ = AuthStore.InsertAdminAuditEvent(c.Request.Context(), &uid, sub, "case.note_added", "operator_case", strconv.FormatInt(caseID, 10),
		reason, rid, c.ClientIP(), c.GetHeader("User-Agent"), meta)
	respond.JSONOK(c, http.StatusCreated, gin.H{"id": noteID, "case_id": caseID})
}

type postSanctionBody struct {
	SanctionType string  `json:"sanction_type" binding:"required"`
	ExpiresAt    *string `json:"expires_at"` // RFC3339 optional
}

// PostOperatorCaseSanction POST /api/v1/cases/:id/sanctions
func PostOperatorCaseSanction(c *gin.Context) {
	uid, ok := middleware.AuthUserIDFromContext(c)
	if !ok {
		respond.Error(c, http.StatusForbidden, api.CodeForbidden, "user subject required", nil)
		return
	}
	caseID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var body postSanctionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid JSON body", nil)
		return
	}
	row, err := AuthStore.GetOperatorCase(c.Request.Context(), caseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "case not found", nil)
			return
		}
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to load case", nil)
		return
	}
	payload := map[string]any{
		"sanction_type":            strings.TrimSpace(body.SanctionType),
		"subject_platform_user_id": row.SubjectPlatformUserID,
	}
	if body.ExpiresAt != nil && strings.TrimSpace(*body.ExpiresAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.ExpiresAt))
		if err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "expires_at must be RFC3339", nil)
			return
		}
		payload["expires_at"] = t.UTC().Format(time.RFC3339)
	}
	reason := reasonFromContext(c)
	_, err = AuthStore.InsertOperatorCaseAction(c.Request.Context(), caseID, authstore.CaseActionSanction, payload, reason, uid)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to record sanction", nil)
		return
	}
	sub := c.GetString("auth_subject")
	rid := requestid.FromContext(c)
	meta, _ := json.Marshal(payload)
	_ = AuthStore.InsertAdminAuditEvent(c.Request.Context(), &uid, sub, "case.sanction_recorded", "operator_case", strconv.FormatInt(caseID, 10),
		reason, rid, c.ClientIP(), c.GetHeader("User-Agent"), meta)
	respond.OK(c, gin.H{"ok": true, "recorded": "sanction"})
}

type postRecoveryBody struct {
	CharacterRef string `json:"character_ref"`
}

// PostOperatorCaseRecoveryRequest POST /api/v1/cases/:id/recovery-requests
func PostOperatorCaseRecoveryRequest(c *gin.Context) {
	uid, ok := middleware.AuthUserIDFromContext(c)
	if !ok {
		respond.Error(c, http.StatusForbidden, api.CodeForbidden, "user subject required", nil)
		return
	}
	caseID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var body postRecoveryBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid JSON body", nil)
		return
	}
	row, err := AuthStore.GetOperatorCase(c.Request.Context(), caseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "case not found", nil)
			return
		}
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to load case", nil)
		return
	}
	cref := strings.TrimSpace(body.CharacterRef)
	if cref == "" && row.SubjectCharacterRef.Valid {
		cref = row.SubjectCharacterRef.String
	}
	payload := map[string]any{
		"character_ref":            cref,
		"subject_platform_user_id": row.SubjectPlatformUserID,
	}
	reason := reasonFromContext(c)
	_, err = AuthStore.InsertOperatorCaseAction(c.Request.Context(), caseID, authstore.CaseActionRecoveryRequest, payload, reason, uid)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to record recovery request", nil)
		return
	}
	sub := c.GetString("auth_subject")
	rid := requestid.FromContext(c)
	meta, _ := json.Marshal(payload)
	_ = AuthStore.InsertAdminAuditEvent(c.Request.Context(), &uid, sub, "case.recovery_requested", "operator_case", strconv.FormatInt(caseID, 10),
		reason, rid, c.ClientIP(), c.GetHeader("User-Agent"), meta)
	respond.OK(c, gin.H{"ok": true, "recorded": "recovery_request"})
}

type postAppealResolveBody struct {
	Outcome string `json:"outcome" binding:"required"` // upheld | overturned
	Notes   string `json:"notes"`
}

// PostOperatorCaseAppealResolve POST /api/v1/cases/:id/appeals/resolve
func PostOperatorCaseAppealResolve(c *gin.Context) {
	uid, ok := middleware.AuthUserIDFromContext(c)
	if !ok {
		respond.Error(c, http.StatusForbidden, api.CodeForbidden, "user subject required", nil)
		return
	}
	caseID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var body postAppealResolveBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid JSON body", nil)
		return
	}
	outcome := strings.ToLower(strings.TrimSpace(body.Outcome))
	if outcome != "upheld" && outcome != "overturned" {
		respond.Error(c, http.StatusBadRequest, api.CodeValidation, "outcome must be upheld or overturned", nil)
		return
	}
	if _, err := AuthStore.GetOperatorCase(c.Request.Context(), caseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "case not found", nil)
			return
		}
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to load case", nil)
		return
	}
	payload := map[string]any{
		"outcome": outcome,
		"notes":   strings.TrimSpace(body.Notes),
	}
	reason := reasonFromContext(c)
	_, err := AuthStore.InsertOperatorCaseAction(c.Request.Context(), caseID, authstore.CaseActionAppealResolve, payload, reason, uid)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to record appeal resolution", nil)
		return
	}
	sub := c.GetString("auth_subject")
	rid := requestid.FromContext(c)
	meta, _ := json.Marshal(payload)
	_ = AuthStore.InsertAdminAuditEvent(c.Request.Context(), &uid, sub, "case.appeal_resolved", "operator_case", strconv.FormatInt(caseID, 10),
		reason, rid, c.ClientIP(), c.GetHeader("User-Agent"), meta)
	respond.OK(c, gin.H{"ok": true, "recorded": "appeal_resolve"})
}

// ListOperatorCaseNotes GET /api/v1/cases/:id/notes
func ListOperatorCaseNotes(c *gin.Context) {
	caseID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	if _, err := AuthStore.GetOperatorCase(c.Request.Context(), caseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "case not found", nil)
			return
		}
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to load case", nil)
		return
	}
	notes, err := AuthStore.ListOperatorCaseNotes(c.Request.Context(), caseID)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to list notes", nil)
		return
	}
	items := make([]gin.H, 0, len(notes))
	for _, n := range notes {
		items = append(items, gin.H{
			"id":                 n.ID,
			"case_id":            n.CaseID,
			"created_at":         n.CreatedAt,
			"body":               n.Body,
			"created_by_user_id": n.CreatedByUserID,
		})
	}
	respond.OK(c, gin.H{"items": items})
}

// ListOperatorCaseActions GET /api/v1/cases/:id/actions
func ListOperatorCaseActions(c *gin.Context) {
	caseID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	if _, err := AuthStore.GetOperatorCase(c.Request.Context(), caseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusNotFound, api.CodeNotFound, "case not found", nil)
			return
		}
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to load case", nil)
		return
	}
	actions, err := AuthStore.ListOperatorCaseActions(c.Request.Context(), caseID, 100)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to list actions", nil)
		return
	}
	items := make([]gin.H, 0, len(actions))
	for _, a := range actions {
		hi := gin.H{
			"id":            a.ID,
			"case_id":       a.CaseID,
			"created_at":    a.CreatedAt,
			"action_kind":   a.ActionKind,
			"actor_user_id": a.ActorUserID,
		}
		if len(a.PayloadJSON) > 0 {
			hi["payload"] = json.RawMessage(append([]byte(nil), a.PayloadJSON...))
		}
		if a.Reason.Valid {
			hi["reason"] = a.Reason.String
		}
		items = append(items, hi)
	}
	respond.OK(c, gin.H{"items": items})
}

func reasonFromContext(c *gin.Context) string {
	reason := strings.TrimSpace(c.GetHeader(middleware.PlatformActionReasonHeader))
	if reason == "" {
		if v, ok := c.Get("platform_action_reason"); ok {
			if s, ok := v.(string); ok {
				reason = strings.TrimSpace(s)
			}
		}
	}
	return reason
}
