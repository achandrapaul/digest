package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/achandrapaul/digest/pkg/system"
)

func fileExists(dir, name string) bool {
	return system.Exists(filepath.Join(dir, name))
}

func InstallCommand(dir string) []string {
	switch {
	case fileExists(dir, "pnpm-lock.yaml"):
		return []string{"pnpm", "install", "--frozen-lockfile", "--ignore-scripts", "--ignore-pnpmfile"}
	case fileExists(dir, "yarn.lock") && fileExists(dir, ".yarnrc.yml"):
		return []string{"yarn", "install", "--immutable", "--mode=skip-build"}
	case fileExists(dir, "yarn.lock"):
		return []string{"yarn", "install", "--frozen-lockfile", "--ignore-scripts"}
	case fileExists(dir, "package-lock.json"):
		return []string{"npm", "ci", "--ignore-scripts"}
	case fileExists(dir, "package.json"):
		return []string{"npm", "install", "--ignore-scripts"}
	}
	return nil
}

var riskyYarnSettings = regexp.MustCompile(`(?m)^\s*(yarnPath|plugins|npmAuthToken|npmAuthIdent)\s*:|^\s*yarn-path\b`)

func riskyInstallConfig(dir string) string {
	if content, err := system.Read(filepath.Join(dir, ".npmrc")); err == nil && strings.Contains(string(content), "${") {
		return ".npmrc expands environment variables"
	}
	for _, name := range []string{".yarnrc.yml", ".yarnrc"} {
		if content, err := system.Read(filepath.Join(dir, name)); err == nil && riskyYarnSettings.Match(content) {
			return name + " sets a yarn binary, plugins or auth"
		}
	}
	return ""
}

var miseExecutable = func() string {
	if path, err := exec.LookPath("mise"); err == nil {
		return path
	}
	if home, err := os.UserHomeDir(); err == nil && system.Exists(filepath.Join(home, ".local", "bin", "mise")) {
		return filepath.Join(home, ".local", "bin", "mise")
	}
	return ""
}

func nvmDir() string {
	if dir := os.Getenv("NVM_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nvm")
}

func nodeVersion(dir string) string {
	content, err := system.Read(filepath.Join(dir, ".nvmrc"))
	if err != nil {
		return ""
	}
	version := strings.TrimPrefix(strings.TrimSpace(string(content)), "v")
	if version == "lts/*" {
		return "lts"
	}
	return version
}

func installScript(dir string) string {
	if riskyInstallConfig(dir) != "" {
		return ""
	}
	install := InstallCommand(dir)
	if len(install) == 0 {
		return ""
	}
	command := strings.Join(install, " ")
	if !fileExists(dir, ".nvmrc") {
		return command
	}
	if nvm := nvmDir(); system.Exists(filepath.Join(nvm, "nvm.sh")) {
		return "export NVM_DIR=" + system.ShellQuote(nvm) + `; . "$NVM_DIR/nvm.sh" && nvm install && ` + command
	}
	if version, mise := nodeVersion(dir), miseExecutable(); version != "" && mise != "" {
		return system.ShellQuote(mise) + " exec " + system.ShellQuote("node@"+version) + " -- " + command
	}
	return command
}
