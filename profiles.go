package config

import (
	"fmt"
	"strings"
)

// ProfileManager manages configuration profiles.
type ProfileManager struct {
	config   *Config
	profiles map[string]map[string]any
	active   string
}

// NewProfileManager creates a new ProfileManager associated with a Config instance.
func NewProfileManager(config *Config) *ProfileManager {
	return &ProfileManager{
		config:   config,
		profiles: make(map[string]map[string]any),
	}
}

// AddProfile adds a named configuration profile.
func (pm *ProfileManager) AddProfile(name string, data map[string]any) {
	pm.profiles[name] = cloneMap(data)
}

// SetActiveProfile switches to a named profile, reloading the configuration.
func (pm *ProfileManager) SetActiveProfile(name string) error {
	if _, exists := pm.profiles[name]; !exists {
		return fmt.Errorf("profile %q does not exist", name)
	}

	pm.active = name
	return pm.applyProfile(name)
}

// GetActiveProfile returns the name of the currently active profile.
func (pm *ProfileManager) GetActiveProfile() string {
	return pm.active
}

// ListProfiles returns a list of all available profile names.
func (pm *ProfileManager) ListProfiles() []string {
	profiles := make([]string, 0, len(pm.profiles))
	for name := range pm.profiles {
		profiles = append(profiles, name)
	}
	return profiles
}

// applyProfile applies a profile's data by adding it as a high-priority source.
func (pm *ProfileManager) applyProfile(name string) error {
	data, exists := pm.profiles[name]
	if !exists {
		return fmt.Errorf("profile %q does not exist", name)
	}

	// Create a temporary memory source with profile data at a very high priority.
	// This ensures it overrides other sources.
	source := MemoryWithPriority(data, 1000)

	// We need to replace the old profile source if it exists.
	pm.config.mu.Lock()
	defer pm.config.mu.Unlock()

	// Remove any existing profile source
	newSources := make([]Source, 0, len(pm.config.sources))
	for _, src := range pm.config.sources {
		if !strings.HasPrefix(src.Name(), "profile:") {
			newSources = append(newSources, src)
		}
	}

	// Add the new profile source
	newSources = append(newSources, source)
	pm.config.sources = newSources
	pm.config.sortSources()

	// Reload the configuration to apply the changes from the new profile source.
	// Note: This is called within a lock, but Load() also tries to lock.
	// To avoid deadlock, we need a private, unlocked version of the loading logic.
	// For simplicity here, we'll release the lock before calling Load().
	// A more robust solution would be to refactor Load() to have an unlocked variant.
	pm.config.mu.Unlock()
	err := pm.config.Load()
	pm.config.mu.Lock() // Re-lock for the function's defer

	return err
}

// LoadProfilesFromConfig scans the loaded configuration for profile definitions
// and an active profile setting.
func (pm *ProfileManager) LoadProfilesFromConfig() error {
	// Look for a 'profiles' key in the configuration
	if profilesData, ok := pm.config.Get("profiles"); ok {
		if profiles, ok := profilesData.(map[string]any); ok {
			for name, data := range profiles {
				if profileData, ok := data.(map[string]any); ok {
					pm.AddProfile(name, profileData)
				}
			}
		}
	}

	// Check for an 'activeProfile' key to set the initial profile
	if activeProfile := pm.config.GetString("activeProfile"); activeProfile != "" {
		return pm.SetActiveProfile(activeProfile)
	}

	return nil
}

// ProfileSource is a dynamic source that loads data from the active profile.
type ProfileSource struct {
	BaseSource
	profileManager *ProfileManager
}

// NewProfileSource creates a new ProfileSource.
func NewProfileSource(profileManager *ProfileManager) *ProfileSource {
	return &ProfileSource{
		BaseSource:     NewBaseSource("profiles", 50), // Medium priority
		profileManager: profileManager,
	}
}

// Load returns the data for the currently active profile.
func (s *ProfileSource) Load() (map[string]any, error) {
	activeProfile := s.profileManager.GetActiveProfile()
	if activeProfile == "" {
		return make(map[string]any), nil
	}

	if profile, exists := s.profileManager.profiles[activeProfile]; exists {
		return cloneMap(profile), nil
	}

	return make(map[string]any), nil
}
