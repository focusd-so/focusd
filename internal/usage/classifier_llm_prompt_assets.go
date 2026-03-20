package usage

import (
	"embed"
	"fmt"
	"strings"
	"sync"
)

//go:embed assets/prompts/*.txt
var promptFS embed.FS

var (
	promptCache     map[string]string
	promptCacheErr  error
	promptCacheOnce sync.Once
)

func loadPromptCache() {
	entries, err := promptFS.ReadDir("assets/prompts")
	if err != nil {
		promptCacheErr = fmt.Errorf("failed to list prompt assets: %w", err)
		return
	}

	cache := make(map[string]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		content, readErr := promptFS.ReadFile("assets/prompts/" + fileName)
		if readErr != nil {
			promptCacheErr = fmt.Errorf("failed to read prompt %q: %w", fileName, readErr)
			return
		}

		prompt := strings.TrimSpace(string(content))
		if prompt == "" {
			promptCacheErr = fmt.Errorf("prompt %q is empty", fileName)
			return
		}

		cache[fileName] = prompt
	}

	promptCache = cache
}

func mustLoadPrompt(kind, fileName string) string {
	promptCacheOnce.Do(loadPromptCache)

	if promptCacheErr != nil {
		panic(fmt.Sprintf("failed to initialize prompts while loading %s prompt %q: %v", kind, fileName, promptCacheErr))
	}

	prompt, ok := promptCache[fileName]
	if !ok {
		panic(fmt.Sprintf("missing %s prompt %q", kind, fileName))
	}

	return prompt
}
