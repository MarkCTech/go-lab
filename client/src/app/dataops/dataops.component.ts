import { Component, OnInit } from '@angular/core';
import { forkJoin, of } from 'rxjs';
import { catchError } from 'rxjs/operators';
import {
  BackupRestoreListData,
  BackupsStatusData,
  BackupRestoreRequestRow,
  SecurityMeData
} from './dataops.models';
import { PlatformService } from '../platform.service';

const PERM_BACKUPS_READ = 'backups.read';
const PERM_RESTORE_REQUEST = 'backups.restore.request';
const PERM_RESTORE_APPROVE = 'backups.restore.approve';
const PERM_RESTORE_FULFILL = 'backups.restore.fulfill';

@Component({
  selector: 'app-dataops',
  templateUrl: './dataops.component.html',
  styleUrls: ['./dataops.component.css']
})
export class DataopsComponent implements OnInit {
  status: BackupsStatusData | null = null;
  list: BackupRestoreListData | null = null;
  me: SecurityMeData | null = null;
  loadError = false;

  /** Create form */
  newScope = '';
  newRestorePoint = '';
  createReason = '';

  /** Shared action reason for row buttons (min 10 server-side) */
  actionReason = '';

  statusFilter = '';

  constructor(private platform: PlatformService) {}

  ngOnInit(): void {
    this.refreshAll();
  }

  refreshAll(): void {
    this.loadError = false;
    forkJoin({
      me: this.platform.getSecurityMeTyped().pipe(catchError(() => of(null))),
      status: this.platform.getBackupsStatus().pipe(catchError(() => of(null))),
      list: this.platform.listBackupRestoreRequests({ limit: 50 }).pipe(catchError(() => of(null)))
    }).subscribe({
      next: ({ me, status, list }) => {
        this.me = me;
        this.status = status;
        this.list = list;
        if (!me || !status || !list) {
          this.loadError = true;
        }
      },
      error: () => {
        this.loadError = true;
      }
    });
  }

  refreshList(): void {
    const st = this.statusFilter.trim();
    this.platform
      .listBackupRestoreRequests({
        limit: 50,
        ...(st ? { status: st } : {})
      })
      .subscribe((d) => {
        if (d) {
          this.list = d;
        }
      });
  }

  hasPerm(perm: string): boolean {
    const p = this.me?.effective_permissions;
    if (!p?.length) {
      return false;
    }
    if (p.includes('*')) {
      return true;
    }
    return p.includes(perm);
  }

  get canSeeBackups(): boolean {
    return this.hasPerm(PERM_BACKUPS_READ);
  }

  get canRequestRestore(): boolean {
    return this.hasPerm(PERM_RESTORE_REQUEST);
  }

  get canApproveRestore(): boolean {
    return this.hasPerm(PERM_RESTORE_APPROVE);
  }

  get canFulfillRestore(): boolean {
    return this.hasPerm(PERM_RESTORE_FULFILL);
  }

  /** Shared action reason for row buttons; server requires ≥ 10 characters. */
  get actionReasonOk(): boolean {
    return this.actionReason.trim().length >= 10;
  }

  submitCreate(): void {
    const reason = this.createReason.trim();
    if (reason.length < 10) {
      return;
    }
    this.platform
      .postBackupRestoreRequest(
        { scope: this.newScope.trim(), restore_point_label: this.newRestorePoint.trim() },
        reason
      )
      .subscribe((res) => {
        if (res) {
          this.newScope = '';
          this.newRestorePoint = '';
          this.createReason = '';
          this.refreshAll();
        }
      });
  }

  private validActionReason(): string | null {
    const r = this.actionReason.trim();
    return r.length >= 10 ? r : null;
  }

  doApprove(row: BackupRestoreRequestRow): void {
    const r = this.validActionReason();
    if (!r) {
      return;
    }
    this.platform.postBackupRestoreApprove(row.id, r).subscribe((res) => {
      if (res) {
        this.refreshAll();
      }
    });
  }

  doReject(row: BackupRestoreRequestRow): void {
    const r = this.validActionReason();
    if (!r) {
      return;
    }
    this.platform.postBackupRestoreReject(row.id, r).subscribe((res) => {
      if (res) {
        this.refreshAll();
      }
    });
  }

  doFulfill(row: BackupRestoreRequestRow): void {
    const r = this.validActionReason();
    if (!r) {
      return;
    }
    this.platform.postBackupRestoreFulfill(row.id, r).subscribe((res) => {
      if (res) {
        this.refreshAll();
      }
    });
  }

  doCancel(row: BackupRestoreRequestRow): void {
    const r = this.validActionReason();
    if (!r) {
      return;
    }
    this.platform.postBackupRestoreCancel(row.id, r).subscribe((res) => {
      if (res) {
        this.refreshAll();
      }
    });
  }

  canApproveRow(row: BackupRestoreRequestRow): boolean {
    if (!this.canApproveRestore || row.status !== 'pending') {
      return false;
    }
    const uid = this.me?.user_id;
    if (uid == null) {
      return false;
    }
    return row.requested_by_user_id !== uid;
  }

  canRejectRow(row: BackupRestoreRequestRow): boolean {
    return this.canApproveRow(row);
  }

  canCancelRow(row: BackupRestoreRequestRow): boolean {
    if (!this.canRequestRestore || row.status !== 'pending') {
      return false;
    }
    return row.requested_by_user_id === this.me?.user_id;
  }

  canFulfillRow(row: BackupRestoreRequestRow): boolean {
    return this.canFulfillRestore && row.status === 'approved';
  }

  trackById(_i: number, row: BackupRestoreRequestRow): number {
    return row.id;
  }
}
