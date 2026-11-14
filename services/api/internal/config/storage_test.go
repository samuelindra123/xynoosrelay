package config

import "testing"

func TestRemoteStorageRequiresCredentials(t *testing.T) {
	for _, provider := range []string{"s3", "r2", "neon"} {
		t.Run(provider, func(t *testing.T) {
			cfg := &Config{AppEnv: EnvDevelopment, StorageProvider: provider}
			if cfg.validate() == nil {
				t.Fatal("accepted remote provider without credentials")
			}
			cfg.StorageBucket = "test"
			cfg.StorageAccessKeyID = "access"
			cfg.StorageSecretAccessKey = "secret"
			if provider == "neon" {
				cfg.StorageEndpoint = "https://storage.example.neon.tech"
				cfg.StorageRegion = "us-east-2"
			}
			if err := cfg.validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
