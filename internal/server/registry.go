package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ParadoxInfinite/oriel/internal/docker"
)

// registryHTTP talks to public registry search APIs that the daemon can't reach
// via `docker search` (which only ever queries Docker Hub).
var registryHTTP = &http.Client{Timeout: 10 * time.Second}

// searchQuay queries Quay.io's public find API and maps each hit to a fully
// qualified, directly-pullable ref (quay.io/<namespace>/<name>).
func searchQuay(ctx context.Context, term string, limit int) ([]docker.SearchResult, error) {
	u := "https://quay.io/api/v1/find/repositories?query=" + url.QueryEscape(term)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := registryHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("quay.io returned %s", resp.Status)
	}

	var body struct {
		Results []struct {
			Kind        string `json:"kind"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Namespace   struct {
				Name string `json:"name"`
			} `json:"namespace"`
		} `json:"results"`
	}
	if err := decodeCapped(resp.Body, &body); err != nil {
		return nil, err
	}

	out := make([]docker.SearchResult, 0, limit)
	for _, r := range body.Results {
		if r.Kind != "repository" || r.Namespace.Name == "" || r.Name == "" {
			continue
		}
		out = append(out, docker.SearchResult{
			Name:        "quay.io/" + r.Namespace.Name + "/" + r.Name,
			Description: cleanDesc(r.Description),
			Official:    r.Namespace.Name == "official",
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// searchECR queries the AWS ECR Public Gallery and maps hits to pullable refs
// (public.ecr.aws/<alias>/<repo>).
func searchECR(ctx context.Context, term string, limit int) ([]docker.SearchResult, error) {
	body, _ := json.Marshal(map[string]string{"searchTerm": term})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.us-east-1.gallery.ecr.aws/searchRepositoryCatalogData", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := registryHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ecr public returned %s", resp.Status)
	}

	var b struct {
		Results []struct {
			RepositoryName string `json:"repositoryName"`
			Alias          string `json:"primaryRegistryAliasName"`
			Description    string `json:"repositoryDescription"`
			Verified       bool   `json:"registryVerified"`
		} `json:"repositoryCatalogSearchResultList"`
	}
	if err := decodeCapped(resp.Body, &b); err != nil {
		return nil, err
	}

	out := make([]docker.SearchResult, 0, limit)
	for _, r := range b.Results {
		if r.RepositoryName == "" || r.Alias == "" {
			continue
		}
		out = append(out, docker.SearchResult{
			Name:        "public.ecr.aws/" + r.Alias + "/" + r.RepositoryName,
			Description: cleanDesc(r.Description),
			Official:    r.Verified,
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// listTags returns recent tags for a repo from a registry that publishes a tag
// API (Docker Hub, Quay). Registries without one return an empty list.
func listTags(ctx context.Context, source, repo string, limit int) ([]string, error) {
	switch source {
	case "quay":
		return tagsQuay(ctx, repo, limit)
	case "dockerhub", "":
		return tagsDockerHub(ctx, repo, limit)
	default:
		return []string{}, nil
	}
}

func tagsDockerHub(ctx context.Context, repo string, limit int) ([]string, error) {
	repo = strings.TrimPrefix(repo, "docker.io/")
	if !strings.Contains(repo, "/") {
		repo = "library/" + repo // official images live under library/
	}
	u := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags?page_size=%d&ordering=last_updated", repo, limit)
	var b struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := getJSON(ctx, u, &b); err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(b.Results))
	for _, r := range b.Results {
		tags = append(tags, r.Name)
	}
	return tags, nil
}

func tagsQuay(ctx context.Context, repo string, limit int) ([]string, error) {
	repo = strings.TrimPrefix(repo, "quay.io/")
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return []string{}, nil
	}
	u := fmt.Sprintf("https://quay.io/api/v1/repository/%s/%s/tag/?limit=%d&onlyActiveTags=true", parts[0], parts[1], limit)
	var b struct {
		Tags []struct {
			Name string `json:"name"`
		} `json:"tags"`
	}
	if err := getJSON(ctx, u, &b); err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(b.Tags))
	for _, t := range b.Tags {
		tags = append(tags, t.Name)
	}
	return tags, nil
}

func getJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := registryHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry returned %s", resp.Status)
	}
	return decodeCapped(resp.Body, v)
}

var htmlTag = regexp.MustCompile(`<[^>]*>`)

// cleanDesc reduces a markdown/HTML README to a single readable line: strip tags,
// then return the first line with real text, capped to a sane length.
func cleanDesc(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(htmlTag.ReplaceAllString(line, ""))
		line = strings.TrimLeft(line, "#>*-= \t")
		if line != "" {
			if len(line) > 160 {
				line = line[:160] + "…"
			}
			return line
		}
	}
	return ""
}
