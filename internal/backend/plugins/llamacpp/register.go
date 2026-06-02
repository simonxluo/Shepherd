package llamacpp

import "github.com/simonxluo/Shepherd/internal/backend"

func init() { backend.MustRegister(New()) }
