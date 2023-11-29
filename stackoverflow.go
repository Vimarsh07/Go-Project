package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	//"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 1 * time.Minute
)

type Question struct {
	ID         int    `gorm:"primary_key" json:"question_id"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	IsAnswered bool   `json:"is_answered"`
	Answers    []Answer
}

type Answer struct {
	ID         int      `gorm:"primary_key" json:"answer_id"`
	QuestionID int      `json:"question_id"`
	Body       string   `json:"body"`
	Question   Question `gorm:"foreignkey:QuestionID"`
}

type APIResponse struct {
	Items          []Question `json:"items"`
	HasMore        bool       `json:"has_more"`
	QuotaMax       int        `json:"quota_max"`
	QuotaRemaining int        `json:"quota_remaining"`
}

var (
	stackOverflowAPICalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stackoverflow_api_calls_questions_persecond",
			Help: "Total number of StackOverflow API calls made(Questions Per Second).",
		},
		[]string{"tag", "days"},
	)
	stackOverflowDataCollected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stackoverflow_data_collected_gigabytes_questions_persecond",
			Help: "Total amount of data collected from StackOverflow in GB(Questions Per Second).",
		},
		[]string{"tag", "days"},
	)
	stackOverflowAPICallsAnswers = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stackoverflow_api_calls_answers_persecond",
			Help: "Total number of StackOverflow API calls made(Answers Per Second).",
		},
		[]string{"tag", "days"},
	)
	stackOverflowDataCollectedAnswers = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stackoverflow_data_collected_gigabytes_answers_persecond",
			Help: "Total amount of data collected from StackOverflow in GB(Answers Per Second).",
		},
		[]string{"tag", "days"},
	)
)

type TwoDaysQuestion struct {
	Question
}

type SevenDaysQuestion struct {
	Question
}

type FortyFiveDaysQuestion struct {
	Question
}

// Table name overrides
func (TwoDaysQuestion) TableName() string {
	return "twodays_questions"
}

func (SevenDaysQuestion) TableName() string {
	return "sevendays_questions"
}

func (FortyFiveDaysQuestion) TableName() string {
	return "fortyfivedays_questions"
}

type TwoDaysAnswer struct {
	Answer
}

type SevenDaysAnswer struct {
	Answer
}

type FortyFiveDaysAnswer struct {
	Answer
}

// Table name overrides
func (TwoDaysAnswer) TableName() string {
	return "twodays_answers"
}

func (SevenDaysAnswer) TableName() string {
	return "sevendays_answers"
}

func (FortyFiveDaysAnswer) TableName() string {
	return "fortyfivedays_answers"
}

func init() {

	prometheus.MustRegister(stackOverflowAPICalls)
	prometheus.MustRegister(stackOverflowDataCollected)
	prometheus.MustRegister(stackOverflowAPICallsAnswers)
	prometheus.MustRegister(stackOverflowDataCollectedAnswers)
}

var db *gorm.DB

func main() {

	username := "postgres"
	password := "root"
	databaseName := "stackoverflow"
	cloudSQLConnectionName := "mercurial-feat-406520:us-central1:mypostgres" // Example: my-project:us-central1:my-instance

	// Construct the DSN (Data Source Name)
	// The DSN for Google Cloud SQL follows the format:
	// "host=/cloudsql/instance-connection-name dbname=your-database-name user=your-database-user password=your-database-password sslmode=disable"
	dsn := fmt.Sprintf("host=/cloudsql/%s dbname=%s user=%s password=%s sslmode=disable",
		cloudSQLConnectionName, databaseName, username, password)

	// Open the database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	db.AutoMigrate(&Question{}, &Answer{}, &TwoDaysQuestion{}, &SevenDaysQuestion{}, &FortyFiveDaysQuestion{}, &TwoDaysAnswer{}, &SevenDaysAnswer{}, &FortyFiveDaysAnswer{})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	tags := []struct {
		Name         string
		MaxQuestions int
	}{
		{"Prometheus", 500},
		{"Selenium", 500},
		{"OpenAI", 500},
		{"Docker", 500},
		{"Milvus", 169}, // Set to the total number of questions available for Milvus
		{"Go", 500},
	}

	for {
		for _, tag := range tags {
			// Fetch for the default timeframe
			fetchQuestionsByTag(tag.Name, tag.MaxQuestions, 0)

			// Fetch for specific timeframes
			fetchQuestionsByTag(tag.Name, tag.MaxQuestions, 2)
			fetchQuestionsByTag(tag.Name, tag.MaxQuestions, 7)
			fetchQuestionsByTag(tag.Name, tag.MaxQuestions, 45)
			// fetchQuestionsByTag(tag.Name, tag.MaxQuestions)
		}

		time.Sleep(24 * time.Hour)
	}

}

// func connectDatabase() *gorm.DB {
// 	username := "postgres"
// 	password := "root"
// 	databaseName := "stackoverflow"
// 	cloudSQLConnectionName := "mercurial-feat-406520:us-central1:mypostgres" // Example: my-project:us-central1:my-instance

// 	// Construct the DSN (Data Source Name)
// 	// The DSN for Google Cloud SQL follows the format:
// 	// "host=/cloudsql/instance-connection-name dbname=your-database-name user=your-database-user password=your-database-password sslmode=disable"
// 	dsn := fmt.Sprintf("host=/cloudsql/%s dbname=%s user=%s password=%s sslmode=disable",
// 		cloudSQLConnectionName, databaseName, username, password)

// 	// Open the database connection
// 	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
// 	if err != nil {
// 		log.Fatalf("Failed to connect to database: %v", err)
// 	}

// 	// Migrate the schema
// 	db.AutoMigrate(&StackOverflowPost{}, &GitHubIssue{})

// 	return db
// }

func fetchQuestionsByTag(tag string, maxQuestions int, daysBack int) {
	apiKey := "NEwms6EZl4*CWqGm1Rxjpg(("
	page := 1
	var hasMore bool = true
	var fetchedQuestionsCount int = 0
	fromDate := time.Now().AddDate(0, 0, -daysBack).Unix()
	daysLabel := fmt.Sprintf("%d_days", daysBack)
	if daysBack == 0 {
		daysLabel = "all"
	}
	for hasMore && fetchedQuestionsCount < maxQuestions {

		fmt.Printf("Fetching page %d for tag: %s\n", page, tag)
		url := fmt.Sprintf("https://api.stackexchange.com/2.3/questions?order=desc&sort=creation&tagged=%s&site=stackoverflow&filter=!9_bDDxJY5&key=%s&pagesize=30&page=%d", tag, apiKey, page)

		if daysBack > 0 {
			url = fmt.Sprintf("https://api.stackexchange.com/2.3/questions?order=desc&sort=creation&tagged=%s&site=stackoverflow&filter=!9_bDDxJY5&key=%s&pagesize=30&page=%d&fromdate=%d", tag, apiKey, page, fromDate)
			// log.Printf("url %s", url)
		}

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error fetching questions for tag %s on page %d: %v", tag, page, err)
			page++
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(time.Second * 10)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Error reading response body for tag %s on page %d: %v", tag, page, err)
			page++
			continue
		}

		stackOverflowAPICalls.WithLabelValues(tag, daysLabel).Inc()
		stackOverflowDataCollected.WithLabelValues(tag, daysLabel).Add(float64(len(body)) / 1e9)

		var apiResponse APIResponse
		if err := json.Unmarshal(body, &apiResponse); err != nil {
			log.Printf("Error parsing JSON for tag %s on page %d: %v", tag, page, err)
			page++
			continue
		}

		for _, question := range apiResponse.Items {
			if fetchedQuestionsCount >= maxQuestions {
				break
			}
			// var existingQuestion Question
			// if db.Where("id = ?", question.ID).First(&existingQuestion).RecordNotFound() {
			switch daysBack {
			case 2:
				db.Create(&TwoDaysQuestion{Question: question})
			case 7:
				db.Create(&SevenDaysQuestion{Question: question})
			case 45:
				db.Create(&FortyFiveDaysQuestion{Question: question})
			default:
				db.Create(&question) // Default case for original table
			}
			// db.Create(&question)
			fetchedQuestionsCount++

			if question.IsAnswered {
				fetchAnswers(question.ID, daysBack)
			}
		}
		// }

		hasMore = apiResponse.HasMore
		page++
	}

	log.Printf("Total questions fetched for tag %s: (%s): %d", tag, daysLabel, fetchedQuestionsCount)
}

func fetchAnswers(questionID int, daysBack int) {
	apiKey := "NEwms6EZl4*CWqGm1Rxjpg(("
	fromDate := time.Now().AddDate(0, 0, -daysBack).Unix()
	url := fmt.Sprintf("https://api.stackexchange.com/2.3/questions/%d/answers?order=desc&sort=activity&site=stackoverflow&filter=!nNPvSNdWme&key=%s", questionID, apiKey)

	if daysBack > 0 {
		url = fmt.Sprintf("https://api.stackexchange.com/2.3/questions/%d/answers?order=desc&sort=activity&site=stackoverflow&filter=!nNPvSNdWme&key=%s&fromdate=%d", questionID, apiKey, fromDate)
	}
	var backoff = initialBackoff

	for {
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error fetching answers for question ID %d: %v", questionID, err)
			return
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			log.Printf("Rate limit exceeded, backing off for %v", backoff)
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
			}
			continue
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body for question ID %d: %v", questionID, err)
			return
		}
		daysLabel := fmt.Sprintf("%d_days", daysBack)
		stackOverflowAPICallsAnswers.WithLabelValues("answers", daysLabel).Inc()
		stackOverflowDataCollectedAnswers.WithLabelValues("answers", daysLabel).Add(float64(len(body)) / 1e9)

		var apiResponse struct {
			Items []Answer `json:"items"`
		}
		if err := json.Unmarshal(body, &apiResponse); err != nil {
			log.Printf("Error parsing JSON for question ID %d: %v", questionID, err)
			return
		}

		for _, answer := range apiResponse.Items {
			// var existingAnswer Answer
			// if db.Where("id = ?", answer.ID).First(&existingAnswer).RecordNotFound() {
			answer.QuestionID = questionID
			if daysBack == 2 {
				db.Create(&TwoDaysAnswer{Answer: answer})
			} else if daysBack == 7 {
				db.Create(&SevenDaysAnswer{Answer: answer})
			} else if daysBack == 45 {
				db.Create(&FortyFiveDaysAnswer{Answer: answer})
			} else {
				db.Create(&answer)
			}
		}
		// }

		return
	}
}

/*func fetchStackOverflowData() []StackOverflowPost {
	var allPosts []StackOverflowPost

	key := "NEwms6EZl4*CWqGm1Rxjpg(("

	// Define the API endpoint with your parameters and include the key

	for _, framework := range frameworks {
		stackoverflowAPICalls.Inc()

		url := fmt.Sprintf("https://api.stackexchange.com/2.3/search/advanced?order=desc&sort=activity&tagged=%s&site=stackoverflow&filter=withbody&key=%s", framework.StackOverflowTag, key)

		resp, err := http.Get(url)
		if err != nil {
			log.Fatalf("Error making request to Stack Overflow API: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error reading response body: %v", err)
		}

		dataCollected.Add(float64(len(body)))

		var result struct {
			Items []StackOverflowPost `json:"items"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Fatalf("Stackover: Error unmarshaling response JSON: %v", err)
		}

		allPosts = append(allPosts, result.Items...)
	}
	return allPosts
}
*/

/*func fetchGitHubData() []GitHubIssue {
	var allIssues []GitHubIssue
	for _, framework := range frameworks {
		githubAPICalls.Inc()

		url := fmt.Sprintf("https://api.github.com/repos/%s/issues", framework.GitHubRepo)
		fmt.Println("Fetching URL:", url) // Log the URL being accessed

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalf("Error creating request: %v", err)
		}

		req.Header.Set("Authorization", "token "+"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")
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error sending request to GitHub API: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Fatalf("API request failed with status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error reading response body: %v", err)
		}

		var issues []GitHubIssue
		if err := json.Unmarshal(body, &issues); err != nil {
			log.Fatalf("Github: Error unmarshaling response JSON: %v", err)
		}

		allIssues = append(allIssues, issues...)
	}
	return allIssues
}

func storeStackOverflowPost(db *gorm.DB, post StackOverflowPost) {
	db.Create(&post)
}

func storeGitHubIssue(db *gorm.DB, issue GitHubIssue) {
	var existingIssue GitHubIssue
	result := db.First(&existingIssue, issue.ID)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		db.Create(&issue)
	} else {
		db.Model(&existingIssue).Updates(issue)
	}
}

func fetchDataAndStore(db *gorm.DB) {

	stackOverflowPosts := fetchStackOverflowData()
	for _, post := range stackOverflowPosts {
		storeStackOverflowPost(db, post)
	}

	gitHubIssues := fetchGitHubData()
	for _, issue := range gitHubIssues {
		storeGitHubIssue(db, issue)
	}
}
*/
