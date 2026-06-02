package vllmomni

import "github.com/simonxluo/Shepherd/internal/backend"

func init() { backend.MustRegister(New()) }
