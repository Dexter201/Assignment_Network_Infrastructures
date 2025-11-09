package feed

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

type FeedHandler struct {
	config *Config
	client *http.Client
}

type Post struct {
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

func createFeedHandler(config *Config) *FeedHandler {
	return &FeedHandler{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second, // Set a timeout for external requests
		},
	}
}

// ServeHTTP is the main entry point for requests to /feed and needs to be implmented to fullfil the handler interface
func (handler *FeedHandler) ServeHTTP(writer http.ResponseWriter, receiver *http.Request) {

	//check if PUT, POST , ... methods are used and nothing else
	if receiver.Method != http.MethodGet {
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//check authorization of user
	userID := receiver.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(writer, "Unauthorized: X-User-ID header missing", http.StatusUnauthorized)
		return
	}

	friendIDs, err := handler.fetchFriends(userID)
	if err != nil {
		log.Printf("Error fetching friends for user %s: %v", userID, err)
		http.Error(writer, "Failed to fetch friends", http.StatusInternalServerError)
		return
	}

	if len(friendIDs) == 0 {
		log.Printf("User %s has no friends, returning empty feed", userID)
		json.NewEncoder(writer).Encode([]Post{}) // Return empty list
		return
	}

	allPosts := handler.fetchPostsForFriends(userID, friendIDs, true)

	sortPostsByTimestamp(allPosts)

	finalFeed := limitPosts(allPosts, 10)

	encodeResponse(writer, receiver, finalFeed)
	log.Printf("Successfully served feed for user %s with %d posts", userID, len(finalFeed))
}

// sortPostsByTimestamp sorts a slice of Posts in place, newest first.
func sortPostsByTimestamp(posts []Post) {
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Timestamp.After(posts[j].Timestamp)
	})
}

// limitPosts returns a slice containing at most 'limit' posts from the input slice.
func limitPosts(posts []Post, limit int) []Post {
	if len(posts) < limit {
		limit = len(posts)
	}
	return posts[:limit]
}

// encodeResponse sets the content type and writes the data as JSON to the ResponseWriter.
func encodeResponse(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (handler *FeedHandler) fetchFriends(userID string) ([]string, error) {
	//build request to fetch friends
	requestURL := fmt.Sprintf("%s/friends", handler.config.UserLBURL)
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create friends request: %w", err)
	}
	// we need to add the userID for the client to be able to authenticate
	request.Header.Set("X-User-ID", userID)

	response, err := handler.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("friends request failed: %w", err)
	}
	defer response.Body.Close() //defer happens at the end of the function --> close the persistant TCP connection

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user-service returned status %d", response.StatusCode)
	}

	var friendIDs []string
	if err := json.NewDecoder(response.Body).Decode(&friendIDs); err != nil {
		return nil, fmt.Errorf("failed to decode friends response: %w", err)
	}

	return friendIDs, nil
}

func (handler *FeedHandler) fetchPosts(authUserID, friendID string) ([]Post, error) {
	// Build the request to fetch the Posts
	requestURL := fmt.Sprintf("%s/posts/%s", handler.config.PostLBURL, friendID)
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create posts request: %w", err)
	}

	request.Header.Set("X-User-ID", authUserID)

	response, err := handler.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("posts request failed for friend %s: %w", friendID, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post-service returned status %d for friend %s", response.StatusCode, friendID)
	}

	var posts []Post
	if err := json.NewDecoder(response.Body).Decode(&posts); err != nil {
		return nil, fmt.Errorf("failed to decode posts response for friend %s: %w", friendID, err)
	}

	return posts, nil
}

func (handler *FeedHandler) fetchPostsForFriends(userID string, friendIDs []string, concurent bool) []Post {
	if concurent {
		return handler.fetchPostsForFriends_Concurrent(userID, friendIDs)
	} else {
		return handler.fetchPostsForFriends_Serial(userID, friendIDs)
	}

}

// fetchPostsForFriends_Concurrent fetches posts for all friends in parallel --> just wondering if concurrency gives us a big performence boost or not.
func (handler *FeedHandler) fetchPostsForFriends_Concurrent(userID string, friendIDs []string) []Post {
	var allPosts []Post
	var wg sync.WaitGroup
	postsChan := make(chan []Post, len(friendIDs))

	for _, friendID := range friendIDs {
		wg.Add(1)
		go func(fID string) {
			defer wg.Done()
			posts, err := handler.fetchPosts(userID, fID)
			if err != nil {
				log.Printf("Failed to fetch posts for friend %s: %v", fID, err)
				return // Don't add posts if there was an error
			}
			postsChan <- posts
		}(friendID)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(postsChan)

	for posts := range postsChan {
		allPosts = append(allPosts, posts...)
	}

	return allPosts
}

// fetchPostsForFriends_Serial fetches posts for all friends serially (one after another)
func (handler *FeedHandler) fetchPostsForFriends_Serial(userID string, friendIDs []string) []Post {
	var allPosts []Post

	// Loop through each friend ID one by one
	for _, friendID := range friendIDs {

		posts, err := handler.fetchPosts(userID, friendID)
		if err != nil {
			log.Printf("Failed to fetch posts for friend %s: %v", friendID, err)
			continue
		}

		allPosts = append(allPosts, posts...)
	}

	return allPosts
}
