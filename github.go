package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	githubAPIURL = "https://api.github.com/repos/%s/%s/issues?page=%d"
	dbConnStr    = "host=/cloudsql/mercurial-feat-406520:us-central1:mypostgres port=5432 user=postgres dbname=github password=root sslmode=disable"
	githubToken  = "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" // Replace with your GitHub token
)

var (
	apiCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "github_api_calls_total_persecond",
			Help: "Total number of GitHub API calls made(PerSecond).",
		},
		[]string{"repository", "days"},
	)
	dataCollected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "github_data_collected_gigabytes_persecond",
			Help: "Total amount of data collected from GitHub in gigabytes(PerSecond).",
		},
		[]string{"repository", "days"},
	)
)

func init() {
	prometheus.MustRegister(apiCalls)
	prometheus.MustRegister(dataCollected)

}

type GitHubIssue struct {
	ID              uint      `gorm:"primary_key"`
	GitHubID        int       `json:"number"`
	Title           string    `json:"title"`
	Body            string    `json:"body"`
	State           string    `json:"state"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	RepositoryOwner string    `gorm:"-"`
	RepositoryName  string    `gorm:"-"`
}

type TwoDaysIssue struct {
	GitHubIssue
}

func (TwoDaysIssue) TableName() string {
	return "twodays"
}

type SevenDaysIssue struct {
	GitHubIssue
}

func (SevenDaysIssue) TableName() string {
	return "sevendays"
}

type FortyFiveDaysIssue struct {
	GitHubIssue
}

func (FortyFiveDaysIssue) TableName() string {
	return "fortyfivedays"
}

func fetchGitHubIssues(db *gorm.DB, owner, repo string, days int) error {
	since := time.Now().AddDate(0, 0, -days).Format(time.RFC3339)
	page := 1
	for {
		start := time.Now()
		url := fmt.Sprintf(githubAPIURL, owner, repo, page)
		if days > 0 {
			url = fmt.Sprintf(githubAPIURL+"&since=%s", owner, repo, page, since)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "token "+githubToken)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch issues: %s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		// apiCalls.WithLabelValues(fmt.Sprintf("%s/%s", owner, repo)).Inc()
		// dataCollected.WithLabelValues(fmt.Sprintf("%s/%s", owner, repo)).Add(float64(len(body)) / 1e9)
		duration := time.Since(start).Seconds() // Measure duration

		if duration == 0 {
			duration = 1 // Avoid division by zero
		}

		bodySize := float64(len(body)) / 1e9 // Size in GB
		daysLabel := fmt.Sprintf("%d", days)

		// Calculate rates
		apiCallsRate := 1 / duration
		dataCollectedRate := bodySize / duration

		apiCalls.WithLabelValues(fmt.Sprintf("%s/%s", owner, repo), daysLabel).Add(apiCallsRate)
		dataCollected.WithLabelValues(fmt.Sprintf("%s/%s", owner, repo), daysLabel).Add(dataCollectedRate)

		var issues []GitHubIssue
		if err := json.Unmarshal(body, &issues); err != nil {
			return err
		}

		for _, issue := range issues {

			if days == 2 {
				db.Create(&TwoDaysIssue{GitHubIssue: issue})
			} else if days == 7 {
				db.Create(&SevenDaysIssue{GitHubIssue: issue})
			} else if days == 45 {
				db.Create(&FortyFiveDaysIssue{GitHubIssue: issue})
			} else {
				db.Create(&issue)

			}
		}

		// Check if there's a next page
		linkHeader := resp.Header.Get("Link")
		if !strings.Contains(linkHeader, `rel="next"`) {
			break
		}
		page++
	}
	return nil
}

func main() {
	db, err := gorm.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	db.AutoMigrate(&GitHubIssue{}, &TwoDaysIssue{}, &SevenDaysIssue{}, &FortyFiveDaysIssue{})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	repositories := []struct {
		Owner string
		Repo  string
	}{
		{"prometheus", "prometheus"},
		{"SeleniumHQ", "selenium"},
		{"openai", "openai-python"},
		{"docker", "docker"},
		{"milvus-io", "milvus"},
		{"golang", "go"},
	}

	for {
		for _, repo := range repositories {
			if err := fetchGitHubIssues(db, repo.Owner, repo.Repo, 0); err != nil {
				log.Printf("Error fetching issues for %s/%s: %v", repo.Owner, repo.Repo, err)
			}
			if err := fetchGitHubIssues(db, repo.Owner, repo.Repo, 2); err != nil {
				log.Printf("Error fetching 2-day issues for %s/%s: %v", repo.Owner, repo.Repo, err)
			}
			if err := fetchGitHubIssues(db, repo.Owner, repo.Repo, 7); err != nil {
				log.Printf("Error fetching 7-day issues for %s/%s: %v", repo.Owner, repo.Repo, err)
			}
			if err := fetchGitHubIssues(db, repo.Owner, repo.Repo, 45); err != nil {
				log.Printf("Error fetching 45-day issues for %s/%s: %v", repo.Owner, repo.Repo, err)
			}
		}
		time.Sleep(24 * time.Hour)
	}
}
