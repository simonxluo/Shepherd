CREATE TABLE conversations (
    id VARCHAR(255) PRIMARY KEY,
    model VARCHAR(255) NOT NULL,
    title TEXT,
    system_prompt TEXT,
    message_count INT DEFAULT 0,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    metadata TEXT
);

CREATE TABLE messages (
    id VARCHAR(255) PRIMARY KEY,
    conversation_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    content LONGTEXT NOT NULL,
    name VARCHAR(255),
    token_count INT DEFAULT 0,
    created_at BIGINT NOT NULL,
    metadata TEXT
);

CREATE TABLE benchmarks (
    id VARCHAR(255) PRIMARY KEY,
    model_id VARCHAR(255) NOT NULL,
    model_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    command TEXT,
    config TEXT,
    metrics TEXT,
    error TEXT,
    created_at BIGINT NOT NULL,
    started_at BIGINT,
    finished_at BIGINT
);

CREATE TABLE benchmark_configs (
    name VARCHAR(255) PRIMARY KEY,
    model_id VARCHAR(255) NOT NULL,
    model_name VARCHAR(255) NOT NULL,
    llamacpp_path TEXT NOT NULL,
    devices TEXT,
    params TEXT,
    created_at BIGINT NOT NULL
);

CREATE TABLE model_load_configs (
    id VARCHAR(255) PRIMARY KEY,
    node_id VARCHAR(255) NOT NULL,
    model_id VARCHAR(255) NOT NULL,
    model_name VARCHAR(255) NOT NULL,
    config TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT '',
    UNIQUE KEY idx_mlc_node_model_name (node_id, model_id, name)
);

CREATE TABLE launch_profiles (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    backend_type VARCHAR(100) NOT NULL,
    installation_id VARCHAR(255),
    model_scope VARCHAR(255),
    params TEXT NOT NULL,
    env TEXT,
    extra_args TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE TABLE model_metadata (
    model_id VARCHAR(255) PRIMARY KEY,
    node_id VARCHAR(255),
    storage_path TEXT,
    alias VARCHAR(255),
    favourite TINYINT DEFAULT 0,
    tags TEXT,
    description TEXT,
    load_count INT DEFAULT 0,
    last_loaded BIGINT,
    total_tokens BIGINT DEFAULT 0,
    capabilities TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE TABLE tts_history (
    id VARCHAR(255) PRIMARY KEY,
    model VARCHAR(255) NOT NULL,
    input_text LONGTEXT NOT NULL,
    audio_path TEXT NOT NULL,
    format VARCHAR(50) NOT NULL,
    duration DOUBLE DEFAULT 0,
    favourite TINYINT DEFAULT 0,
    params TEXT,
    created_at BIGINT NOT NULL
);

CREATE TABLE download_tasks (
    id VARCHAR(255) PRIMARY KEY,
    url TEXT NOT NULL,
    path TEXT NOT NULL,
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    state VARCHAR(50) NOT NULL DEFAULT 'idle',
    downloaded_bytes BIGINT DEFAULT 0,
    total_bytes BIGINT DEFAULT 0,
    etag VARCHAR(255) DEFAULT '',
    range_supported TINYINT DEFAULT 0,
    final_url TEXT,
    temp_file_name VARCHAR(255) DEFAULT '',
    parts_total INT DEFAULT 0,
    parts_completed INT DEFAULT 0,
    file_type VARCHAR(100) DEFAULT '',
    source_type VARCHAR(100) DEFAULT '',
    repo_id VARCHAR(255) DEFAULT '',
    error_message TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 5,
    created_at BIGINT NOT NULL,
    started_at BIGINT DEFAULT 0,
    finished_at BIGINT DEFAULT 0
);
