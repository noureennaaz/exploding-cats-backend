package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "github.com/redis/go-redis/v9"
    "sort"
    "os"
    
)

var (
    redisClient *redis.Client
)

type User struct {
    Username string `json:"username"`
    Points   int    `json:"points"`
}

func main() {

    opt, err := redis.ParseURL(os.Getenv("CONNECTION_STRING"))
    if err != nil {
        panic(err)
    }
    
   
    redisClient = redis.NewClient(opt)

    http.HandleFunc("/", handler)
    http.HandleFunc("/register-user", registerUserHandler)
    http.HandleFunc("/leaderboard", leaderboardHandler)
    http.HandleFunc("/register-win", IncrementPointsHandler)
    
    // Runnnig the server
    port := os.Getenv("PORT")
    if port == "" {
        port = "3000"
    }

    fmt.Println("Server running on port", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        fmt.Printf("Failed to start server: %v\n", err)
    }
}

func handler(w http.ResponseWriter, r *http.Request) {
    
    w.WriteHeader(http.StatusCreated)
    fmt.Printf("hit successfull" )
    
}

func registerUserHandler(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Reading the request body
    var newUser User
    if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
        http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
        return
    }

    // Checking if user already presen
    userExists, err := redisClient.Exists(context.Background(), newUser.Username).Result()
    if err != nil {
        http.Error(w, "Error checking user existence", http.StatusInternalServerError)
        return
    }

    if userExists == 1 {
        fmt.Fprintf(w, "User logged in successfully")
        return
    }


    newUser.Points = 0

    //converting to json
    userJSON, err := json.Marshal(newUser)
    if err != nil {
        http.Error(w, "Error registering user", http.StatusInternalServerError)
        return
    }

    // adding redis
    if err := redisClient.Set(context.Background(), newUser.Username, userJSON, 0).Err(); err != nil {
        http.Error(w, "Error registering user", http.StatusInternalServerError)
        return
    }

    // Return success response
    w.WriteHeader(http.StatusCreated)
    fmt.Fprintf(w, "User registered successfully")
}


func leaderboardHandler(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    
    ctx := context.Background()
    keys, err := redisClient.Keys(ctx, "*").Result()
    if err != nil {
        http.Error(w, "Failed to retrieve user data", http.StatusInternalServerError)
        return
    }

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

    
    sort.Slice(leaderboard, func(i, j int) bool {
        return leaderboard[i].Points > leaderboard[j].Points
    })

    
    w.Header().Set("Content-Type", "application/json")

    w.WriteHeader(http.StatusOK)

    encoder := json.NewEncoder(w)

    if err := encoder.Encode(leaderboard); err != nil {
        http.Error(w, "Failed to generate leaderboard", http.StatusInternalServerError)
        return
    }
}
func IncrementPointsHandler(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Reading request body
    var reqBody struct {
        Username string `json:"username"`
    }
    if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
        http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
        return
    }

    // Retrieving user data from Redis
    userJSON, err := redisClient.Get(context.Background(), reqBody.Username).Result()
    if err != nil {
        http.Error(w, "Failed to retrieve user data", http.StatusInternalServerError)
        return
    }

    // json to user struct
    var user User
    if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
        http.Error(w, "Failed to parse user data", http.StatusInternalServerError)
        return
    }

    user.Points++

    //    to json
    updatedUserJSON, err := json.Marshal(user)
    if err != nil {
        http.Error(w, "Failed to update user data", http.StatusInternalServerError)
        return
    }

    // Updating user data 
    if err := redisClient.Set(context.Background(), user.Username, updatedUserJSON, 0).Err(); err != nil {
        http.Error(w, "Failed to update user data", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Points incremented successfully"))
}

