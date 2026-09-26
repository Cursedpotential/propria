//! Byline: Claude Code · Opus 5 · 2026-09-17
//!
//! The donor's command registry, copied verbatim from `apps/src-tauri/src/main.rs`
//! (`tauri::generate_handler!` at main.rs:284). Regenerate from that list when the
//! donor adds commands; the engine adds its own commands in `engine_commands()`.
#![allow(unused_imports)]

use xplorer::agent;
use xplorer::agent_sessions;
use xplorer::ai;
use xplorer::duplicate_finder;
use xplorer::extensions;
use xplorer::file_organizer;
use xplorer::file_watcher;
use xplorer::git;
use xplorer::google_drive;
use xplorer::mcp_host;
use xplorer::mcp_server;
use xplorer::operations;
use xplorer::pty;
use xplorer::shortcuts;
use xplorer::storage;
use xplorer::audit_log;
use xplorer::backup;
use xplorer::file_versions;
use xplorer::sync;

pub fn donor_commands() -> Vec<(&'static str, tauri::CommandFn)> {
    tauri::generate_handler![
            // File system operations (modular)
            operations::read_directory,
            operations::read_text_file,
            operations::extract_document_text,
            operations::read_binary_file,
            operations::get_file_properties,
            operations::get_file_meta_data,
            operations::copy,
            operations::copy_with_progress,
            operations::cancel_file_operation,
            operations::accelerated_copy_file,
            operations::accelerated_copy_directory,
            operations::move_file,
            operations::move_with_progress,
            operations::rename,
            operations::remove_dir,
            operations::remove_file,
            operations::move_to_trash,
            operations::open_recycle_bin,
            operations::get_trash_items,
            operations::restore_trash_item,
            operations::empty_recycle_bin,
            operations::permanently_delete_trash_item,
            // Properties operations
            operations::get_detailed_file_properties,
            operations::get_directory_size,
            operations::get_directory_item_count,
            operations::set_file_permissions,
            // AI Agent operations
            operations::agent_read_file_tree,
            operations::agent_request_write_permission,
            operations::agent_write_file_with_permission,
            operations::is_dir,
            operations::get_files_in_directory,
            operations::open_url,
            operations::open_file,
            operations::file_exist,
            operations::create_dir_recursive,
            operations::create_file,
            operations::create_file_with_content,
            operations::bulk_rename,
            operations::open_in_terminal,
            operations::execute_command,
            operations::execute_command_stream,
            operations::get_current_shell,
            operations::find_files,
            operations::search_in_files,
            operations::grep_search,
            operations::get_app_version,
            operations::install_cli,
            operations::show_in_folder,
            operations::list_drives,
            operations::eject_volume,
            operations::get_dir_size,
            // AI operations
            ai::get_ai_models,
            ai::check_ollama_status,
            xplorer::filesystem_index::filesystem_index_search,
            ai::chat_with_ai,
            ai::analyze_file_with_ai,
            ai::get_file_help,
            operations::folder_ops::calculate_folder_size,
            operations::folder_ops::get_cached_folder_sizes,
            operations::folder_ops::clear_folder_size_cache,
            ai::get_user_directories,
            ai::get_recent_folders,
            ai::add_to_recent_folders,
            ai::get_system_info,
            // AI-powered rename & auto-tag
            ai::suggest_filename,
            ai::auto_tag_files,
            // Undo/Redo
            operations::undo_redo::undo_operation,
            operations::undo_redo::redo_operation,
            operations::undo_redo::get_undo_history,
            operations::undo_redo::clear_undo_history,
            // File templates
            operations::file_ops::get_file_templates,
            operations::file_ops::create_from_template,
            // Conflict resolution
            operations::file_ops::check_conflicts,
            operations::file_ops::get_rename_destination,
            // Disk usage treemap
            operations::file_ops::get_directory_sizes,
            // Symlink creation
            operations::file_ops::create_symlink,
            // Native plugin invoke (for extensions with native code)
            extensions::native_plugin_invoke,
            // Search engine v2 (replaces old tokenizer commands)
            xplorer::search::compat_commands::set_tokenizer_settings,
            xplorer::search::compat_commands::get_tokenizer_settings,
            xplorer::search::compat_commands::rebuild_token_index,
            xplorer::search::compat_commands::search_tokens,
            xplorer::search::compat_commands::natural_language_search,
            xplorer::search::compat_commands::get_tokenizer_stats,
            xplorer::search::compat_commands::is_tokenizer_indexing,
            xplorer::search::compat_commands::get_file_tokens,
            xplorer::search::compat_commands::add_path_to_tokenizer,
            xplorer::search::compat_commands::get_file_recommendations,
            xplorer::search::compat_commands::parse_search_query,
            xplorer::search::compat_commands::enhanced_search,
            // Auto-index and context-aware search
            xplorer::search::compat_commands::index_directory,
            xplorer::search::compat_commands::set_search_context,
            xplorer::search::compat_commands::add_whitelisted_path,
            xplorer::search::compat_commands::ai_search,
            xplorer::search::compat_commands::smart_search,
            // AI indexing (new search engine v2 pipeline)
            xplorer::search::compat_commands::get_ai_index_status,
            xplorer::search::compat_commands::trigger_ai_indexing,
            xplorer::search::compat_commands::get_ai_index_entry,
            // Semantic / hybrid search (new search engine v2 pipeline)
            xplorer::search::compat_commands::semantic_search,
            xplorer::search::compat_commands::find_similar_files,
            xplorer::search::compat_commands::hybrid_search,
            // Shortcut operations
            shortcuts::get_shortcuts,
            shortcuts::get_shortcuts_by_category,
            shortcuts::add_shortcut,
            shortcuts::update_shortcut,
            shortcuts::remove_shortcut,
            shortcuts::get_shortcut_profiles,
            shortcuts::switch_shortcut_profile,
            shortcuts::get_shortcut_settings,
            shortcuts::update_shortcut_settings,
            shortcuts::execute_shortcut_action,
            shortcuts::register_extension_shortcut,
            shortcuts::unregister_extension_shortcuts,
            shortcuts::register_global_shortcuts,
            shortcuts::unregister_global_shortcuts,
            shortcuts::toggle_global_shortcuts,
            shortcuts::reset_shortcuts,
            shortcuts::reset_single_shortcut,
            // Duplicate finder operations
            duplicate_finder::find_duplicates,
            duplicate_finder::cancel_duplicate_scan,
            duplicate_finder::delete_duplicate_files,
            duplicate_finder::move_duplicate_files_to_trash,
            // Extension operations (from extensions module)
            extensions::get_installed_extensions,
            extensions::install_extension_from_path,
            extensions::uninstall_extension_by_id,
            extensions::activate_extension,
            extensions::deactivate_extension,
            extensions::get_extension_permissions,
            extensions::get_active_extension_ids,
            extensions::validate_extension_path,
            extensions::download_and_install_extension,
            extensions::check_for_extension_updates,
            extensions::download_extension,
            extensions::check_extension_updates,
            extensions::pack_extension,
            extensions::marketplace_proxy,
            extensions::install_xtension_file,
            extensions::inspect_xtension_file,
            // WASM backend operations
            extensions::extension_backend_call,
            extensions::extension_backend_status,
            // Git history operations
            git::status::find_git_repository,
            git::status::get_repository_info,
            git::history::get_file_history,
            git::history::get_file_blame,
            git::history::get_file_diff,
            git::history::get_commit_diff,
            git::status::get_file_status,
            git::branch_ops::get_branches,
            git::branch_ops::create_branch,
            git::branch_ops::switch_branch,
            git::branch_ops::delete_branch,
            git::staging::stage_file,
            git::staging::unstage_file,
            git::staging::commit_changes,
            git::stash_ops::get_stashes,
            git::stash_ops::create_stash,
            git::stash_ops::apply_stash,
            git::stash_ops::drop_stash,
            git::history::get_all_commits,
            // Git remote operations
            git::remote_ops::git_pull,
            git::remote_ops::git_push,
            git::remote_ops::git_fetch,
            git::remote_ops::git_get_remotes,
            // File comparison operations
            operations::compute_file_hash,
            operations::compare_files,
            // File association operations
            operations::get_file_associations,
            operations::open_file_with_application,
            operations::get_system_applications,
            operations::set_default_application,
            // Compression operations
            operations::compress_files,
            operations::get_compression_info,
            // Extraction operations
            operations::extract_archive,
            operations::extract_selected_entries,
            operations::get_archive_info,
            operations::is_archive,
            // File organizer operations
            file_organizer::analyze_directory,
            file_organizer::preview_organization,
            file_organizer::execute_organization,
            // Claude Agent operations
            agent::agent_chat,
            agent::agent_respond_approval,
            agent::agent_cancel_session,
            agent::get_agent_settings,
            agent::update_agent_settings,
            agent::update_agent_api_keys,
            agent::agent_approve_plan,
            agent::agent_get_plan,
            agent::get_agent_memory,
            agent::clear_agent_memory,
            agent::delete_agent_memory,
            agent::get_agent_permissions,
            agent::update_agent_permissions,
            // MCP Host — tool provider for external AI clients
            mcp_host::mcp_list_tools,
            mcp_host::mcp_call_tool,
            // Agent session management
            agent_sessions::create_agent_session,
            agent_sessions::list_agent_sessions,
            agent_sessions::get_agent_session,
            agent_sessions::stop_agent_session,
            agent_sessions::remove_agent_session,
            // Recent files operations
            storage::add_recent_file,
            storage::get_recent_files,
            storage::clear_recent_files,
            storage::remove_recent_file,
            // Bookmark operations
            storage::get_bookmarks,
            storage::add_bookmark,
            storage::remove_bookmark,
            storage::update_bookmark_name,
            // File tags operations
            storage::get_file_tags,
            storage::set_file_tags,
            storage::get_all_file_tags,
            storage::get_file_tags_batch,
            storage::remove_all_tags_from_file,
            storage::remove_tag_globally,
            // Batch tag operations
            storage::batch_add_tags,
            storage::batch_remove_tags,
            // Notes operations
            storage::get_file_notes,
            storage::add_file_note,
            storage::update_file_note,
            storage::delete_file_note,
            storage::get_all_notes,
            storage::search_notes,
            // Batch notes operations
            storage::batch_set_notes,
            // Annotation operations
            storage::get_file_annotations,
            storage::add_file_annotation,
            storage::toggle_annotation_resolved,
            storage::delete_file_annotation,
            storage::get_all_annotations,
            // Tag category operations
            storage::get_tag_categories,
            storage::add_tag_category,
            storage::update_tag_category,
            storage::delete_tag_category,
            // Custom metadata operations
            storage::get_file_metadata,
            storage::set_file_metadata,
            storage::get_all_metadata_keys,
            // Chat history operations
            storage::get_chat_sessions,
            storage::get_chat_session,
            storage::save_chat_session,
            storage::delete_chat_session,
            storage::clear_chat_history,
            // Extension-scoped storage operations
            storage::get_extension_storage,
            storage::set_extension_storage,
            storage::delete_extension_storage,
            // Chat-as-Files operations
            storage::chat_files::get_chats_directory,
            storage::chat_files::create_chat_file,
            storage::chat_files::read_chat_file,
            storage::chat_files::save_chat_file,
            storage::chat_files::get_chat_file_summary,
            // Storage analytics operations
            operations::analyze_storage,
            // Directory diagnostics
            operations::diagnose_directory,
            // Encryption operations
            operations::encrypt_file,
            operations::decrypt_file,
            operations::is_encrypted_file,
            // Secure delete operations
            operations::secure_delete,
            // Image processing operations
            operations::resize_image,
            operations::convert_image,
            operations::get_image_info,
            operations::batch_process_images,
            operations::rotate_image,
            operations::flip_image,
            operations::crop_image,
            operations::adjust_brightness,
            operations::adjust_contrast,
            operations::grayscale_image,
            // Google Drive operations
            google_drive::gdrive_authenticate,
            google_drive::gdrive_disconnect,
            google_drive::gdrive_list_accounts,
            google_drive::gdrive_list_files,
            google_drive::gdrive_download_file,
            google_drive::gdrive_upload_file,
            google_drive::gdrive_delete_file,
            google_drive::gdrive_rename_file,
            google_drive::gdrive_move_file,
            google_drive::gdrive_create_folder,
            google_drive::gdrive_get_file_content,
            google_drive::get_gdrive_settings,
            google_drive::update_gdrive_settings,
            // File watcher operations (primary + multi-watcher)
            file_watcher::start_watching,
            file_watcher::stop_watching,
            file_watcher::watch_directory,
            file_watcher::unwatch_directory,
            file_watcher::get_active_watchers,
            // SQLite database operations
            operations::database_ops::list_sqlite_tables,
            operations::database_ops::get_sqlite_table_columns,
            operations::database_ops::query_sqlite_table,
            operations::database_ops::execute_sqlite_query,
            // Git integration (consolidated into git module)
            git::status::get_git_status,
            git::status::get_git_repo_info,
            // Audit log operations
            audit_log::get_audit_log,
            audit_log::clear_audit_log,
            audit_log::export_audit_log,
            // Backup operations
            backup::create_backup,
            backup::list_backups,
            backup::restore_backup,
            backup::delete_backup,
            // Docker operations
            operations::docker_ops::docker_is_available,
            operations::docker_ops::docker_list_containers,
            operations::docker_ops::docker_list_images,
            operations::docker_ops::docker_start_container,
            operations::docker_ops::docker_stop_container,
            operations::docker_ops::docker_remove_container,
            operations::docker_ops::docker_remove_image,
            operations::docker_ops::docker_container_logs,
            // Shell integration operations (Windows registry)
            operations::set_default_folder_handler,
            operations::add_context_menu_entry,
            operations::remove_context_menu_entry,
            operations::get_shell_integration_status,
            // File versioning operations
            file_versions::enable_versioning,
            file_versions::disable_versioning,
            file_versions::create_version,
            file_versions::list_versions,
            file_versions::restore_version,
            file_versions::delete_version,
            file_versions::get_versioning_config,
            file_versions::update_versioning_config,
            file_versions::is_versioning_enabled,
            file_versions::get_version_count,
            file_versions::delete_all_versions,
            file_versions::read_version_content,
            // PTY (interactive terminal) operations
            pty::pty_spawn,
            pty::pty_write,
            pty::pty_resize,
            pty::pty_kill,
            pty::pty_kill_all,
            // Cloud sync operations
            sync::sync_bookmarks_to_cloud,
            sync::sync_tags_to_cloud,
            sync::fetch_cloud_bookmarks,
            sync::fetch_cloud_tags,
            sync::set_last_sync_time,
            sync::get_last_sync_time,
            sync::sync_all,
            sync::auto_sync_on_startup,
            sync::start_auto_sync,
            sync::stop_auto_sync,
            sync::get_auto_sync_status,
    ]
}
