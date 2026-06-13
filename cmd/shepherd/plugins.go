package main

// Side-effect imports that register backend plugins at process start.
import (
	_ "github.com/simonxluo/Shepherd/internal/backend/plugins/llamacpp"
	_ "github.com/simonxluo/Shepherd/internal/backend/plugins/vllm"
	_ "github.com/simonxluo/Shepherd/internal/backend/plugins/vllmomni"
)
