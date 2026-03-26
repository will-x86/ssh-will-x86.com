package content

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Project struct {
	ProjectTitle   string `json:"title"`
	ProjectContent string `json:"content"`
	ProjectNumber  int    `json:"number"`
}

// bubbles/list.Item interface.
func (p Project) Title() string { return fmt.Sprintf("%d. %s", p.ProjectNumber, p.ProjectTitle) }
func (p Project) Description() string {
	if len(p.ProjectContent) > 100 {
		return p.ProjectContent[:100] + "..."
	}
	return p.ProjectContent
}
func (p Project) FilterValue() string { return p.ProjectTitle }

func LoadProjects() ([]Project, error) {
	data, err := os.ReadFile("projects.txt")
	if err != nil {
		return nil, err
	}

	blocks := strings.Split(string(data), "---")
	var projects []Project

	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}

		lines := strings.Split(strings.TrimSpace(block), "\n")
		var p Project
		var contentLines []string

		for i, line := range lines {
			line = strings.TrimSpace(line)
			if t, found := strings.CutPrefix(line, "Title:"); found {
				p.ProjectTitle = strings.TrimSpace(t)
			} else if numStr, found := strings.CutPrefix(line, "Number:"); found {
				num, _ := strconv.Atoi(strings.TrimSpace(numStr))
				p.ProjectNumber = num
			} else if line != "" || i > 2 {
				contentLines = append(contentLines, line)
			}
		}
		p.ProjectContent = strings.TrimSpace(strings.Join(contentLines, "\n"))
		if p.ProjectTitle != "" {
			projects = append(projects, p)
		}
	}

	return projects, nil
}
