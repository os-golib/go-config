package config

// SourceFactory creates sources with consistent patterns and priorities.
type SourceFactory struct {
	defaultPriority int
}

// NewSourceFactory creates a new SourceFactory with a given default priority.
func NewSourceFactory(defaultPriority int) *SourceFactory {
	return &SourceFactory{defaultPriority: defaultPriority}
}

// CreateMemorySource creates a memory source with the factory's default priority.
func (f *SourceFactory) CreateMemorySource(data map[string]any) Source {
	return MemoryWithPriority(data, f.defaultPriority)
}

// CreateFileSource creates a file source with the factory's default priority.
func (f *SourceFactory) CreateFileSource(path string) Source {
	return FileWithPriority(path, f.defaultPriority)
}

// CreateEnvSource creates an environment source with the factory's default priority.
func (f *SourceFactory) CreateEnvSource(prefix string) Source {
	return EnvWithPriority(prefix, f.defaultPriority)
}

// CreateMultiFileSource creates a multi-file source with the factory's default priority.
func (f *SourceFactory) CreateMultiFileSource(pattern string) Source {
	return GlobWithPriority(pattern, f.defaultPriority)
}

// CreateSourceFromType creates a source based on a type string, auto-detecting if necessary.
func (f *SourceFactory) CreateSourceFromType(sourceType, path string, data map[string]any) Source {
	switch sourceType {
	case "memory":
		return f.CreateMemorySource(data)
	case "file":
		return f.CreateFileSource(path)
	case "env":
		return f.CreateEnvSource(path)
	case "glob":
		return f.CreateMultiFileSource(path)
	default:
		// Auto-detect file type by pattern or extension
		if path == "" {
			return f.CreateMemorySource(data)
		}
		// Check for glob patterns
		for _, char := range path {
			if char == '*' || char == '?' || char == '[' {
				return f.CreateMultiFileSource(path)
			}
		}
		// Default to a single file source
		return f.CreateFileSource(path)
	}
}
