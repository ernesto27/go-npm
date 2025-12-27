package scripts

type TrustChecker struct {
	trustedDependencies []string
}

func NewTrustChecker(trustedDeps []string) *TrustChecker {
	return &TrustChecker{
		trustedDependencies: trustedDeps,
	}
}

func (tc *TrustChecker) IsTrusted(packageName string) bool {
	for _, trusted := range tc.trustedDependencies {
		if trusted == packageName {
			return true
		}
	}
	return false
}

func (tc *TrustChecker) SetTrustedDependencies(trustedDeps []string) {
	tc.trustedDependencies = trustedDeps
}
