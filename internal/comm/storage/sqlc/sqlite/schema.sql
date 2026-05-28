CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    model TEXT NOT NULL,
    title TEXT,
    system_prompt TEXT,
    message_count INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    metadata TEXT
);

CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    name TEXT,
    token_count INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    metadata TEXT,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE TABLE benchmarks (
    id TEXT PRIMARY KEY,
    model_id TEXT NOT NULL,
    model_name TEXT NOT NULL,
    status TEXT NOT NULL,
    command TEXT,
    config TEXT,
    metrics TEXT,
    error TEXT,
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER
);

CREATE TABLE benchmark_configs (
    name TEXT PRIMARY KEY,
    model_id TEXT NOT NULL,
    model_name TEXT NOT NULL,
    llamacpp_path TEXT NOT NULL,
    devices TEXT,
    params TEXT,
    created_at INTEGER NOT NULL
);

CREATE TABLE model_load_configs (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    model_id TEXT NOT NULL,
    model_name TEXT NOT NULL,
    config TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    name TEXT NOT NULL DEFAULT ''
);

CREATE TABLE launch_profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    backend_type TEXT NOT NULL,
    installation_id TEXT,
    model_scope TEXT,
    params TEXT NOT NULL,
    env TEXT,
    extra_args TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE model_metadata (
    model_id TEXT PRIMARY KEY,
    node_id TEXT,
    storage_path TEXT,
    alias TEXT,
    favourite INTEGER DEFAULT 0,
    tags TEXT,
    description TEXT,
    load_count INTEGER DEFAULT 0,
    last_loaded INTEGER,
    total_tokens INTEGER DEFAULT 0,
    capabilities TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE tts_history (
    id TEXT PRIMARY KEY,
    model TEXT NOT NULL,
    input_text TEXT NOT NULL,
    audio_path TEXT NOT NULL,
    format TEXT NOT NULL,
    duration REAL DEFAULT 0,
    favourite INTEGER DEFAULT 0,
    params TEXT,
    created_at INTEGER NOT NULL
);

CREATE TABLE download_tasks (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    path TEXT NOT NULL,
    file_name TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL DEFAULT 'idle',
    downloaded_bytes INTEGER DEFAULT 0,
    total_bytes INTEGER DEFAULT 0,
    etag TEXT DEFAULT '',
    range_supported INTEGER DEFAULT 0,
    final_url TEXT DEFAULT '',
    temp_file_name TEXT DEFAULT '',
    parts_total INTEGER DEFAULT 0,
    parts_completed INTEGER DEFAULT 0,
    file_type TEXT DEFAULT '',
    source_type TEXT DEFAULT '',
    repo_id TEXT DEFAULT '',
    error_message TEXT DEFAULT '',
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 5,
    created_at INTEGER NOT NULL,
    started_at INTEGER DEFAULT 0,
    finished_at INTEGER DEFAULT 0
);

CREATE UNIQUE INDEX idx_mlc_node_model_name ON model_load_configs(node_id, model_id, name);
CREATE INDEX idx_messages_conversation ON messages(conversation_id);
CREATE INDEX idx_conversations_created ON conversations(created_at);
CREATE INDEX idx_conversations_updated ON conversations(updated_at);
CREATE INDEX idx_benchmarks_model_id ON benchmarks(model_id);
CREATE INDEX idx_benchmarks_status ON benchmarks(status);
CREATE INDEX idx_benchmarks_created ON benchmarks(created_at);
CREATE INDEX idx_launch_profiles_backend ON launch_profiles(backend_type);
CREATE INDEX idx_launch_profiles_model_scope ON launch_profiles(model_scope);
CREATE INDEX idx_tts_history_created ON tts_history(created_at);
CREATE INDEX idx_tts_history_favourite ON tts_history(favourite);
CREATE INDEX idx_download_tasks_state ON download_tasks(state);
CREATE INDEX idx_download_tasks_created ON download_tasks(created_at);
