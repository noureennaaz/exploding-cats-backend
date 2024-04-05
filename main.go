package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "github.com/redis/go-redis/v9"
    "sort"
    
)

var (
    redisClient *redis.Client
)

type User struct {
    Username string `json:"username"`
    Points   int    `json:"points"`
}

func main() {

    opt, err := redis.ParseURL("redis://default:ifmv7KSW0H6yjDUV9bpbSOsvMtNDwLZc@redis-10045.c100.us-east-1-4.ec2.cloud.redislabs.com:10045")
    if err != nil {
        panic(err)
    }
   
    redisClient = redis.NewClient(opt)

    // Define HTTP endpoints
 
    http.HandleFunc("/register-user", registerUserHandler)
    http.HandleFunc("/leaderboard", leaderboardHandler)
    http.HandleFunc("/register-win", IncrementPointsHandler)
    
    // Run the server
    fmt.Println("Server running on port 8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        fmt.Printf("Failed to start server: %v\n", err)
    }
}

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Read request body
    var newUser User
    if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
        http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
        return
    }

    // Check if the user already exists
    userExists, err := redisClient.Exists(context.Background(), newUser.Username).Result()
    if err != nil {
        http.Error(w, "Error checking user existence", http.StatusInternalServerError)
        return
    }

    if userExists == 1 {
        fmt.Fprintf(w, "User logged in successfully")
        return
    }

    // Set initial points for the new user
    newUser.Points = 0

    // Marshal the user data to JSON
    userJSON, err := json.Marshal(newUser)
    if err != nil {
        http.Error(w, "Error registering user", http.StatusInternalServerError)
        return
    }

    // Add the new user to Redis
    if err := redisClient.Set(context.Background(), newUser.Username, userJSON, 0).Err(); err != nil {
        http.Error(w, "Error registering user", http.StatusInternalServerError)
        return
    }

    // Return success response
    w.WriteHeader(http.StatusCreated)
    fmt.Fprintf(w, "User registered successfully")
}


func leaderboardHandler(w http.ResponseWriter, r *http.Request) {
    // Get all keys (usernames) from Redis
    ctx := context.Background()
    keys, err := redisClient.Keys(ctx, "*").Result()
    if err != nil {
        http.Error(w, "Failed to retrieve user data", http.StatusInternalServerError)
        return
    }

    // Get user data for each key (username)
    var leaderboard []User
    for _, key := range keys {
        userJSON, err := redisClient.Get(ctx, key).Result()
        if err != nil {
            http.Error(w, "Failed to retrieve user data", http.StatusInternalServerError)
            return
        }
        var user User
        if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
            http.Error(w, "Failed to parse user data", http.StatusInternalServerError)
            return
        }
        leaderboard = append(leaderboard, user)
    }

    // Sort the leaderboard based on points (descending order)
    sort.Slice(leaderboard, func(i, j int) bool {
        return leaderboard[i].Points > leaderboard[j].Points
    })

    // Set the Content-Type header to indicate JSON content
    w.Header().Set("Content-Type", "application/json")

    // Write the status code (200 OK) to the response
    w.WriteHeader(http.StatusOK)

    // Create a JSON encoder for the response writer
    encoder := json.NewEncoder(w)

    // Encode the leaderboard data directly to the response writer
    if err := encoder.Encode(leaderboard); err != nil {
        http.Error(w, "Failed to generate leaderboard", http.StatusInternalServerError)
        return
    }
}
func IncrementPointsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Read request body
    var reqBody struct {
        Username string `json:"username"`
    }
    if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
        http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
        return
    }

    // Retrieve user data from Redis
    userJSON, err := redisClient.Get(context.Background(), reqBody.Username).Result()
    if err != nil {
        http.Error(w, "Failed to retrieve user data", http.StatusInternalServerError)
        return
    }

    // Unmarshal user data into User struct
    var user User
    if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
        http.Error(w, "Failed to parse user data", http.StatusInternalServerError)
        return
    }

    // Increment user's points
    user.Points++

    // Marshal updated user data
    updatedUserJSON, err := json.Marshal(user)
    if err != nil {
        http.Error(w, "Failed to update user data", http.StatusInternalServerError)
        return
    }

    // Update user data in Redis
    if err := redisClient.Set(context.Background(), user.Username, updatedUserJSON, 0).Err(); err != nil {
        http.Error(w, "Failed to update user data", http.StatusInternalServerError)
        return
    }

    // Return success response
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Points incremented successfully"))
}



