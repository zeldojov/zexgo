package email

import "strings"

type Sender interface {
	Send(to, subject, body string) error
}

type Service struct {
	sender Sender
}

func NewService(sender Sender) *Service {
	return &Service{
		sender: sender,
	}
}

// The above package declaration and import should be removed as they are duplicates.

func IsAllowedDomain(address string, allowedDomains ...string) bool {
	_, domain, ok := strings.Cut(address, "@")
	if !ok {
		return false
	}

	for _, allowedDomain := range allowedDomains {
		if strings.EqualFold(domain, allowedDomain) {
			return true
		}
	}

	return false
}

func Validate(address string) bool {
	if len(address) == 0 || len(address) > 254 {
		return false
	}

	if strings.Count(address, "@") != 1 {
		return false
	}

	local, domain, _ := strings.Cut(address, "@")

	if local == "" || domain == "" {
		return false
	}

	if local[0] == '.' || local[len(local)-1] == '.' {
		return false
	}

	if strings.Contains(local, "..") {
		return false
	}

	if domain[0] == '.' || domain[len(domain)-1] == '.' {
		return false
	}

	if strings.Contains(domain, "..") {
		return false
	}

	for i := 0; i < len(local); i++ {
		if !isEmailChar(local[i]) {
			return false
		}
	}

	for i := 0; i < len(domain); i++ {
		if !isDomainChar(domain[i]) {
			return false
		}
	}

	return true
}

func isEmailChar(c byte) bool {
	return isASCIILetter(c) ||
		c >= '0' && c <= '9' ||
		c == '.' ||
		c == '+'
}

func isDomainChar(c byte) bool {
	return isASCIILetter(c) ||
		c >= '0' && c <= '9' ||
		c == '.'
}

func isASCIILetter(c byte) bool {
	return c >= 'a' && c <= 'z' ||
		c >= 'A' && c <= 'Z'
}
