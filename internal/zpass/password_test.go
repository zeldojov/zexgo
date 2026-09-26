package zpass

import (
	"errors"
	"strings"
	"testing"
)

func TestPasswordHashVerify(t *testing.T) {
	password := "CorrectHorseBatteryStaple123!"

	encodedHash, err := NewPasswordHash(password)
	if err != nil {
		t.Fatalf("NewPasswordHash() error = %v", err)
	}

	tests := []struct {
		name        string
		password    string
		encodedHash PasswordHash
		want        bool
	}{
		{
			name:        "valid password",
			password:    password,
			encodedHash: encodedHash,
			want:        true,
		},
		{
			name:        "wrong password",
			password:    "WrongPassword123!",
			encodedHash: encodedHash,
			want:        false,
		},
		{
			name:        "invalid number of parts",
			password:    password,
			encodedHash: PasswordHash("$argon2id$v=19$m=65536,t=3,p=2$abc"),
			want:        false,
		},
		{
			name:     "invalid algorithm",
			password: password,
			encodedHash: PasswordHash(strings.Replace(
				string(encodedHash),
				"$argon2id$",
				"$argon2i$",
				1,
			)),
			want: false,
		},
		{
			name:     "invalid version",
			password: password,
			encodedHash: PasswordHash(strings.Replace(
				string(encodedHash),
				"$v=19$",
				"$v=18$",
				1,
			)),
			want: false,
		},
		{
			name:     "invalid parameters format",
			password: password,
			encodedHash: PasswordHash(
				"$argon2id$v=19$invalid$YWJjZGVmZ2hpamtsbW5vcA$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo",
			),
			want: false,
		},
		{
			name:     "invalid memory",
			password: password,
			encodedHash: PasswordHash(strings.Replace(
				string(encodedHash),
				"m=65536",
				"m=32768",
				1,
			)),
			want: false,
		},
		{
			name:     "invalid iterations",
			password: password,
			encodedHash: PasswordHash(strings.Replace(
				string(encodedHash),
				"t=3",
				"t=2",
				1,
			)),
			want: false,
		},
		{
			name:     "invalid parallelism",
			password: password,
			encodedHash: PasswordHash(strings.Replace(
				string(encodedHash),
				"p=2",
				"p=1",
				1,
			)),
			want: false,
		},
		{
			name:     "invalid salt base64",
			password: password,
			encodedHash: func() PasswordHash {
				parts := strings.Split(string(encodedHash), "$")
				parts[4] = "!!!"
				return PasswordHash(strings.Join(parts, "$"))
			}(),
			want: false,
		},
		{
			name:     "invalid salt length",
			password: password,
			encodedHash: func() PasswordHash {
				parts := strings.Split(string(encodedHash), "$")
				parts[4] = "YWJj"
				return PasswordHash(strings.Join(parts, "$"))
			}(),
			want: false,
		},
		{
			name:     "invalid hash base64",
			password: password,
			encodedHash: func() PasswordHash {
				parts := strings.Split(string(encodedHash), "$")
				parts[5] = "!!!"
				return PasswordHash(strings.Join(parts, "$"))
			}(),
			want: false,
		},
		{
			name:     "invalid hash length",
			password: password,
			encodedHash: func() PasswordHash {
				parts := strings.Split(string(encodedHash), "$")
				parts[5] = "YWJj"
				return PasswordHash(strings.Join(parts, "$"))
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.encodedHash.Verify(tt.password)

			if got != tt.want {
				t.Errorf(
					"PasswordHash.Verify() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestNewPasswordHash(t *testing.T) {
	password := "CorrectHorseBatteryStaple123!"

	hash, err := NewPasswordHash(password)
	if err != nil {
		t.Fatalf("NewPasswordHash() error = %v", err)
	}

	if hash == "" {
		t.Fatal("NewPasswordHash() returned empty hash")
	}

	if !strings.HasPrefix(string(hash), "$argon2id$v=19$") {
		t.Errorf("NewPasswordHash() hash has invalid prefix: %q", hash)
	}

	if !strings.Contains(string(hash), "m=65536,t=3,p=2") {
		t.Errorf("NewPasswordHash() hash has incorrect parameters: %q", hash)
	}

	parts := strings.Split(string(hash), "$")
	if len(parts) != 6 {
		t.Fatalf(
			"NewPasswordHash() produced %d parts, want 6: %q",
			len(parts),
			hash,
		)
	}

	if parts[4] == "" {
		t.Error("NewPasswordHash() produced empty salt")
	}

	if parts[5] == "" {
		t.Error("NewPasswordHash() produced empty hash")
	}
}

func TestNewPasswordHashDifferentHashes(t *testing.T) {
	password := "CorrectHorseBatteryStaple123!"

	hash1, err := NewPasswordHash(password)
	if err != nil {
		t.Fatalf("NewPasswordHash() error = %v", err)
	}

	hash2, err := NewPasswordHash(password)
	if err != nil {
		t.Fatalf("NewPasswordHash() error = %v", err)
	}

	if hash1 == hash2 {
		t.Error("NewPasswordHash() produced identical hashes for the same password")
	}
}

func TestNewPasswordHashRandError(t *testing.T) {
	original := randRead
	defer func() {
		randRead = original
	}()

	randRead = func([]byte) (int, error) {
		return 0, errors.New("random error")
	}

	hash, err := NewPasswordHash("CorrectHorseBatteryStaple123!")

	if err == nil {
		t.Fatal("NewPasswordHash() error = nil, want error")
	}

	if err.Error() != "random error" {
		t.Errorf(
			"NewPasswordHash() error = %q, want %q",
			err.Error(),
			"random error",
		)
	}

	if hash != "" {
		t.Errorf("NewPasswordHash() hash = %q, want empty PasswordHash", hash)
	}
}
