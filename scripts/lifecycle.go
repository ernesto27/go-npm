package scripts

import (
	"fmt"
)

type LifecycleManager struct {
	executor        *ScriptExecutor
	trustChecker    *TrustChecker
	nodeModulesPath string
	ignoreScripts   bool
}

func NewLifecycleManager(nodeModulesPath string, ignoreScripts bool) *LifecycleManager {
	return &LifecycleManager{
		executor:        NewScriptExecutor(nodeModulesPath),
		trustChecker:    NewTrustChecker([]string{}),
		nodeModulesPath: nodeModulesPath,
		ignoreScripts:   ignoreScripts,
	}
}

func (lm *LifecycleManager) SetTrustedDependencies(trustedDeps []string) {
	lm.trustChecker.SetTrustedDependencies(trustedDeps)
}

func (lm *LifecycleManager) RunPackageScripts(pkgName, pkgVersion, pkgPath string, scripts any) error {
	return lm.runPackageScripts(pkgName, pkgVersion, pkgPath, scripts, true)
}

func (lm *LifecycleManager) RunRootPackageScripts(pkgName, pkgVersion, pkgPath string, scripts any) error {
	return lm.runPackageScripts(pkgName, pkgVersion, pkgPath, scripts, false)
}

func (lm *LifecycleManager) runPackageScripts(pkgName, pkgVersion, pkgPath string, scripts any, checkTrust bool) error {
	if lm.ignoreScripts {
		return nil
	}

	if checkTrust && !lm.trustChecker.IsTrusted(pkgName) {
		return nil
	}

	scriptMap := extractScripts(scripts)
	if scriptMap == nil {
		return nil
	}

	if preinstall, exists := scriptMap["preinstall"]; exists {
		if err := lm.executor.Execute(preinstall, pkgPath, pkgName, pkgVersion, "preinstall"); err != nil {
			return fmt.Errorf("preinstall script failed for %s: %w", pkgName, err)
		}
	}

	if install, exists := scriptMap["install"]; exists {
		if err := lm.executor.Execute(install, pkgPath, pkgName, pkgVersion, "install"); err != nil {
			return fmt.Errorf("install script failed for %s: %w", pkgName, err)
		}
	}

	if postinstall, exists := scriptMap["postinstall"]; exists {
		if err := lm.executor.Execute(postinstall, pkgPath, pkgName, pkgVersion, "postinstall"); err != nil {
			return fmt.Errorf("postinstall script failed for %s: %w", pkgName, err)
		}
	}

	return nil
}

func (lm *LifecycleManager) RunPrepare(pkgName, pkgVersion, rootPath string, scripts any) error {
	if lm.ignoreScripts {
		return nil
	}

	scriptMap := extractScripts(scripts)
	if scriptMap == nil {
		return nil
	}

	prepare, exists := scriptMap["prepare"]
	if !exists {
		return nil
	}

	if err := lm.executor.Execute(prepare, rootPath, pkgName, pkgVersion, "prepare"); err != nil {
		return fmt.Errorf("prepare script failed: %w", err)
	}

	return nil
}

func extractScripts(scripts any) map[string]string {
	if scripts == nil {
		return nil
	}

	if m, ok := scripts.(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range m {
			if str, ok := v.(string); ok {
				result[k] = str
			}
		}
		return result
	}

	if m, ok := scripts.(map[string]string); ok {
		return m
	}

	return nil
}
