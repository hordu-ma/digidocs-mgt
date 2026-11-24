BEGIN;

CREATE UNIQUE INDEX uq_data_folder_root_name ON data_folders (project_id, name) WHERE parent_id IS NULL;
ALTER TABLE projects ADD CONSTRAINT uq_projects_id_team UNIQUE (id, team_space_id);
ALTER TABLE data_folders ADD CONSTRAINT uq_data_folders_id_project UNIQUE (id, project_id);
ALTER TABLE data_assets ADD CONSTRAINT fk_asset_project_team
    FOREIGN KEY (project_id, team_space_id) REFERENCES projects (id, team_space_id) NOT VALID;
ALTER TABLE data_assets ADD CONSTRAINT fk_asset_folder_project
    FOREIGN KEY (folder_id, project_id) REFERENCES data_folders (id, project_id) NOT VALID;

INSERT INTO schema_migrations (version, name) VALUES (17, '017_asset_scope_integrity');
COMMIT;
