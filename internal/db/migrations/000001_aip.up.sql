-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
-- SPDX-License-Identifier: Apache-2.0
CREATE TABLE aip (
  id BIGINT NOT NULL AUTO_INCREMENT,
  uuid CHAR(36) NOT NULL,
  name VARCHAR(2048) NOT NULL,
  status ENUM('queued','processing','completed','error','canceled') NOT NULL DEFAULT 'queued',
  created_at DATETIME NOT NULL,
  profile ENUM('files_and_metadata','files_metadata_and_manifests','original_and_extracted') NOT NULL DEFAULT 'files_and_metadata',
  ocfl_object_key VARCHAR(1024) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY aip_uuid (uuid),
  KEY aip_status_id (status,id),
  KEY aip_created_at (created_at)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
