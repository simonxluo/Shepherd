-- +goose Up
UPDATE launch_profiles SET backend_type = 'llamacpp' WHERE backend_type IN ('llama.cpp', '');
UPDATE launch_profiles SET backend_type = 'vllmomni' WHERE backend_type = 'vllm_omni';

-- +goose Down
UPDATE launch_profiles SET backend_type = 'llama.cpp' WHERE backend_type = 'llamacpp';
UPDATE launch_profiles SET backend_type = 'vllm_omni' WHERE backend_type = 'vllmomni';
