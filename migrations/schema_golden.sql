
/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;
DROP TABLE IF EXISTS `admin_audit_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_audit_events` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `actor_user_id` int DEFAULT NULL,
  `auth_subject` varchar(255) NOT NULL,
  `action` varchar(128) NOT NULL,
  `resource_type` varchar(64) NOT NULL,
  `resource_id` varchar(128) DEFAULT NULL,
  `reason` text,
  `request_id` varchar(128) DEFAULT NULL,
  `ip` varchar(45) DEFAULT NULL,
  `user_agent` varchar(512) DEFAULT NULL,
  `meta_json` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_admin_audit_created` (`created_at`),
  KEY `idx_admin_audit_actor` (`actor_user_id`),
  KEY `idx_admin_audit_action` (`action`),
  CONSTRAINT `fk_admin_audit_actor` FOREIGN KEY (`actor_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `auth_audit_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `auth_audit_events` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `event_type` varchar(64) NOT NULL,
  `user_id` int DEFAULT NULL,
  `ip` varchar(45) DEFAULT NULL,
  `user_agent` varchar(512) DEFAULT NULL,
  `subject_hint` varchar(255) DEFAULT NULL,
  `meta_json` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_auth_audit_created` (`created_at`),
  KEY `idx_auth_audit_user` (`user_id`),
  KEY `idx_auth_audit_type` (`event_type`)
) ENGINE=InnoDB AUTO_INCREMENT=19 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `auth_desktop_exchange_codes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `auth_desktop_exchange_codes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `code_hash` char(64) NOT NULL COMMENT 'SHA-256 hex of one-time desktop exchange code',
  `code_challenge` varchar(128) NOT NULL COMMENT 'PKCE-style S256 challenge for code_verifier proof',
  `session_id` varchar(128) NOT NULL,
  `callback_uri` varchar(512) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expires_at` timestamp NOT NULL,
  `consumed_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_auth_desktop_exchange_code_hash` (`code_hash`),
  KEY `idx_auth_desktop_exchange_user` (`user_id`),
  KEY `idx_auth_desktop_exchange_expires` (`expires_at`),
  KEY `idx_auth_desktop_exchange_consumed` (`consumed_at`),
  CONSTRAINT `fk_auth_desktop_exchange_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `auth_refresh_tokens`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `auth_refresh_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `session_id` bigint unsigned NOT NULL,
  `token_hash` char(64) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expires_at` timestamp NOT NULL,
  `revoked_at` timestamp NULL DEFAULT NULL,
  `replaced_by_id` bigint unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_auth_refresh_token_hash` (`token_hash`),
  KEY `fk_auth_refresh_user` (`user_id`),
  KEY `idx_auth_refresh_session` (`session_id`),
  CONSTRAINT `fk_auth_refresh_session` FOREIGN KEY (`session_id`) REFERENCES `auth_sessions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_auth_refresh_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `auth_sessions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `auth_sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `token_hash` char(64) NOT NULL COMMENT 'SHA-256 hex of opaque session token',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_seen_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expires_at` timestamp NOT NULL,
  `absolute_expires_at` timestamp NOT NULL,
  `revoked_at` timestamp NULL DEFAULT NULL,
  `ip` varchar(45) DEFAULT NULL,
  `user_agent` varchar(512) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_auth_sessions_token_hash` (`token_hash`),
  KEY `idx_auth_sessions_user` (`user_id`),
  KEY `idx_auth_sessions_revoked` (`revoked_at`),
  CONSTRAINT `fk_auth_sessions_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `backup_restore_requests`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `backup_restore_requests` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `requested_by_user_id` int NOT NULL,
  `scope` varchar(64) NOT NULL,
  `restore_point_label` varchar(256) NOT NULL,
  `reason` text NOT NULL,
  `status` varchar(32) NOT NULL DEFAULT 'pending',
  `rejection_reason` text,
  `approval_1_user_id` int DEFAULT NULL,
  `approval_1_at` timestamp NULL DEFAULT NULL,
  `approval_2_user_id` int DEFAULT NULL,
  `approval_2_at` timestamp NULL DEFAULT NULL,
  `fulfilled_at` timestamp NULL DEFAULT NULL,
  `fulfilled_note` text,
  `request_id` varchar(128) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_backup_restore_status` (`status`),
  KEY `idx_backup_restore_requested_by` (`requested_by_user_id`),
  KEY `idx_backup_restore_created` (`created_at`),
  KEY `fk_backup_restore_approval_1` (`approval_1_user_id`),
  KEY `fk_backup_restore_approval_2` (`approval_2_user_id`),
  CONSTRAINT `fk_backup_restore_approval_1` FOREIGN KEY (`approval_1_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_backup_restore_approval_2` FOREIGN KEY (`approval_2_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_backup_restore_requested_by` FOREIGN KEY (`requested_by_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `economy_ledger_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `economy_ledger_events` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `platform_user_id` int NOT NULL,
  `event_type` varchar(64) NOT NULL,
  `amount_delta` bigint NOT NULL DEFAULT '0',
  `currency_code` varchar(32) NOT NULL DEFAULT 'default',
  `reference_type` varchar(64) DEFAULT NULL,
  `reference_id` varchar(128) DEFAULT NULL,
  `meta_json` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_economy_ledger_created` (`created_at`),
  KEY `idx_economy_ledger_user_created` (`platform_user_id`,`created_at`),
  KEY `idx_economy_ledger_event_type` (`event_type`),
  CONSTRAINT `fk_economy_ledger_user` FOREIGN KEY (`platform_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `operator_account_roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `operator_account_roles` (
  `operator_account_id` bigint unsigned NOT NULL,
  `role_id` smallint unsigned NOT NULL,
  `assigned_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`operator_account_id`,`role_id`),
  KEY `idx_operator_account_roles_role` (`role_id`),
  CONSTRAINT `fk_operator_account_roles_account` FOREIGN KEY (`operator_account_id`) REFERENCES `operator_accounts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_operator_account_roles_role` FOREIGN KEY (`role_id`) REFERENCES `platform_roles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `operator_accounts`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `operator_accounts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `linked_user_id` int NOT NULL,
  `email` varchar(255) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `status` varchar(32) NOT NULL DEFAULT 'active',
  `invited_by_user_id` int DEFAULT NULL,
  `last_login_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_operator_accounts_linked_user` (`linked_user_id`),
  UNIQUE KEY `uq_operator_accounts_email` (`email`),
  KEY `idx_operator_accounts_status` (`status`),
  KEY `idx_operator_accounts_invited_by` (`invited_by_user_id`),
  CONSTRAINT `fk_operator_accounts_invited_by` FOREIGN KEY (`invited_by_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_operator_accounts_linked_user` FOREIGN KEY (`linked_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `operator_case_actions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `operator_case_actions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `case_id` bigint unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `action_kind` varchar(32) NOT NULL,
  `payload_json` json DEFAULT NULL,
  `reason` text,
  `actor_user_id` int NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_operator_case_actions_case` (`case_id`),
  KEY `idx_operator_case_actions_kind` (`action_kind`),
  KEY `fk_operator_case_actions_actor` (`actor_user_id`),
  CONSTRAINT `fk_operator_case_actions_actor` FOREIGN KEY (`actor_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_operator_case_actions_case` FOREIGN KEY (`case_id`) REFERENCES `operator_cases` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `operator_case_notes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `operator_case_notes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `case_id` bigint unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `body` text NOT NULL,
  `created_by_user_id` int NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_operator_case_notes_case` (`case_id`),
  KEY `fk_operator_case_notes_author` (`created_by_user_id`),
  CONSTRAINT `fk_operator_case_notes_author` FOREIGN KEY (`created_by_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_operator_case_notes_case` FOREIGN KEY (`case_id`) REFERENCES `operator_cases` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `operator_cases`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `operator_cases` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `status` varchar(32) NOT NULL DEFAULT 'open',
  `priority` varchar(16) NOT NULL DEFAULT 'normal',
  `subject_platform_user_id` int NOT NULL,
  `subject_character_ref` varchar(128) DEFAULT NULL,
  `title` varchar(256) NOT NULL,
  `description` text,
  `created_by_user_id` int NOT NULL,
  `assigned_to_user_id` int DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_operator_cases_status` (`status`),
  KEY `idx_operator_cases_subject_user` (`subject_platform_user_id`),
  KEY `idx_operator_cases_created` (`created_at`),
  KEY `fk_operator_cases_created_by` (`created_by_user_id`),
  KEY `fk_operator_cases_assigned_to` (`assigned_to_user_id`),
  CONSTRAINT `fk_operator_cases_assigned_to` FOREIGN KEY (`assigned_to_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_operator_cases_created_by` FOREIGN KEY (`created_by_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_operator_cases_subject_user` FOREIGN KEY (`subject_platform_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `operator_invites`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `operator_invites` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `token_hash` char(64) NOT NULL COMMENT 'SHA-256 hex of one-time invite token',
  `email` varchar(255) NOT NULL,
  `display_name` varchar(100) NOT NULL,
  `role_name` varchar(64) NOT NULL,
  `linked_user_id` int DEFAULT NULL,
  `invited_by_user_id` int DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expires_at` timestamp NOT NULL,
  `consumed_at` timestamp NULL DEFAULT NULL,
  `meta_json` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_operator_invites_token_hash` (`token_hash`),
  KEY `idx_operator_invites_email` (`email`),
  KEY `idx_operator_invites_expires` (`expires_at`),
  KEY `idx_operator_invites_consumed` (`consumed_at`),
  KEY `idx_operator_invites_role_name` (`role_name`),
  KEY `fk_operator_invites_linked_user` (`linked_user_id`),
  KEY `fk_operator_invites_invited_by` (`invited_by_user_id`),
  CONSTRAINT `fk_operator_invites_invited_by` FOREIGN KEY (`invited_by_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_operator_invites_linked_user` FOREIGN KEY (`linked_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `platform_roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `platform_roles` (
  `id` smallint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_platform_roles_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `user_identities`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_identities` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `issuer` varchar(512) NOT NULL,
  `subject` varchar(512) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_user_identities_issuer_subject` (`issuer`(255),`subject`(255)),
  KEY `idx_user_identities_user` (`user_id`),
  CONSTRAINT `fk_user_identities_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `user_platform_roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_platform_roles` (
  `user_id` int NOT NULL,
  `role_id` smallint unsigned NOT NULL,
  `assigned_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_id`,`role_id`),
  KEY `idx_user_platform_roles_role` (`role_id`),
  CONSTRAINT `fk_user_platform_roles_role` FOREIGN KEY (`role_id`) REFERENCES `platform_roles` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_platform_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `email` varchar(255) DEFAULT NULL,
  `password_hash` varchar(255) DEFAULT NULL,
  `pennies` int NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_email` (`email`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

