package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGetProductsHonorsContextCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-r.Context().Done()
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := client.GetProducts(ctx, ListOptions{})
		result <- err
	}()

	<-requestStarted
	cancel()

	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("GetProducts() error = %v, want context.Canceled", err)
	}
}

func TestGetProductsAppliesListOptions(t *testing.T) {
	requestReceived := make(chan url.Values, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived <- r.URL.Query()
		fmt.Fprint(w, `{"products":[],"total":0,"skip":20,"limit":5}`)
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, HTTPClient: server.Client()}
	_, err := client.GetProducts(context.Background(), ListOptions{
		Limit:  5,
		Skip:   20,
		SortBy: "price",
		Order:  "DESC",
		Select: []string{"title", "price"},
	})
	if err != nil {
		t.Fatal(err)
	}

	query := <-requestReceived
	for key, want := range map[string]string{
		"limit":  "5",
		"skip":   "20",
		"sortBy": "price",
		"order":  "desc",
		"select": "title,price",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("query %q = %q, want %q", key, got, want)
		}
	}
}

func TestGetUsersKeepsDefaultLimitWithoutOptions(t *testing.T) {
	requestReceived := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived <- r.URL.Query().Get("limit")
		fmt.Fprint(w, `{"users":[],"total":0,"skip":0,"limit":10}`)
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, HTTPClient: server.Client()}
	if _, err := client.GetUsers(context.Background(), ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := <-requestReceived; got != "10" {
		t.Errorf("default limit = %q, want 10", got)
	}
}

func TestListEndpointRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name    string
		options ListOptions
	}{
		{name: "negative limit", options: ListOptions{Limit: -1}},
		{name: "negative skip", options: ListOptions{Skip: -1}},
		{name: "invalid field", options: ListOptions{SortBy: "price&order=asc"}},
		{name: "invalid order", options: ListOptions{SortBy: "price", Order: "sideways"}},
		{name: "order without sort field", options: ListOptions{Order: "asc"}},
		{name: "empty selected field", options: ListOptions{Select: []string{"title", " "}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := listEndpoint("/products", test.options); err == nil {
				t.Fatal("listEndpoint() error = nil, want validation error")
			}
		})
	}
}

func TestGetJSONIncludesHTTPErrorDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "record not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, HTTPClient: server.Client()}
	_, err := client.GetProductByID(context.Background(), 123)
	if err == nil {
		t.Fatal("GetProductByID() error = nil, want HTTP error")
	}
	for _, want := range []string{"/products/123", "404 Not Found", "record not found"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

func TestGetPostsAndTodosByUserID(t *testing.T) {
	requestReceived := make(chan *http.Request, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived <- r
		switch r.URL.Path {
		case "/users/7/posts":
			fmt.Fprint(w, `{"posts":[{"id":11,"title":"Post","userId":7}],"total":1,"skip":2,"limit":3}`)
		case "/todos/user/7":
			fmt.Fprint(w, `{"todos":[{"id":21,"todo":"Learn Go","completed":false,"userId":7}],"total":1,"skip":0,"limit":10}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, HTTPClient: server.Client()}
	posts, err := client.GetPostsByUserID(context.Background(), 7, ListOptions{Limit: 3, Skip: 2})
	if err != nil {
		t.Fatal(err)
	}
	todos, err := client.GetTodosByUserID(context.Background(), 7, ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts.Posts) != 1 || posts.Posts[0].ID != 11 {
		t.Errorf("posts = %+v, want post ID 11", posts.Posts)
	}
	if len(todos.Todos) != 1 || todos.Todos[0].ID != 21 || todos.Todos[0].Completed {
		t.Errorf("todos = %+v, want incomplete todo ID 21", todos.Todos)
	}

	requests := map[string]*http.Request{}
	for range 2 {
		request := <-requestReceived
		requests[request.URL.Path] = request
	}
	if request := requests["/users/7/posts"]; request == nil {
		t.Error("posts request path = missing /users/7/posts")
	} else if request.URL.Query().Get("limit") != "3" || request.URL.Query().Get("skip") != "2" {
		t.Errorf("posts query = %s, want limit=3 and skip=2", request.URL.RawQuery)
	}
	if requests["/todos/user/7"] == nil {
		t.Error("todos request path = missing /todos/user/7")
	}
}

func TestMissingResourceEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/search":
			fmt.Fprint(w, `{"users":[{"id":7,"firstName":"Jane","username":"jane"}],"total":1,"skip":2,"limit":5}`)
		case "/products/category/smartphones":
			fmt.Fprint(w, `{"products":[{"id":1,"title":"Phone","category":"smartphones"}],"total":1,"skip":0,"limit":1}`)
		case "/users/7/posts":
			fmt.Fprint(w, `{"posts":[{"id":11,"title":"Post","userId":7}],"total":1,"skip":0,"limit":2}`)
		case "/todos":
			fmt.Fprint(w, `{"todos":[{"id":2,"todo":"Learn Go","completed":false,"userId":7}],"total":1,"skip":0,"limit":3}`)
		case "/todos/2":
			fmt.Fprint(w, `{"id":2,"todo":"Learn Go","completed":false,"userId":7}`)
		case "/todos/user/7":
			fmt.Fprint(w, `{"todos":[{"id":2,"todo":"Learn Go","completed":false,"userId":7}],"total":1,"skip":0,"limit":4}`)
		case "/recipes":
			fmt.Fprint(w, `{"recipes":[{"id":1,"name":"Pizza","ingredients":["dough"],"instructions":["bake"],"mealType":["Dinner"]}],"total":1,"skip":0,"limit":2}`)
		case "/recipes/1":
			fmt.Fprint(w, `{"id":1,"name":"Pizza","ingredients":["dough"],"instructions":["bake"],"mealType":["Dinner"]}`)
		case "/recipes/search":
			fmt.Fprint(w, `{"recipes":[{"id":1,"name":"Pizza"}],"total":1,"skip":0,"limit":2}`)
		case "/comments":
			fmt.Fprint(w, `{"comments":[
				{"id":1,"body":"from target","postId":5,"likes":2,"user":{"id":7,"username":"jane"}},
				{"id":2,"body":"other author","postId":5,"likes":10,"user":{"id":8,"username":"alex"}},
				{"id":3,"body":"also target","postId":6,"likes":9,"user":{"id":7,"username":"jane"}}
			],"total":3,"skip":0,"limit":0}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, HTTPClient: server.Client()}
	ctx := context.Background()

	users, err := client.SearchUsers(ctx, "Jane Doe", ListOptions{Limit: 5, Skip: 2})
	if err != nil || len(users.Users) != 1 || users.Users[0].Username != "jane" {
		t.Fatalf("SearchUsers() = %+v, %v", users, err)
	}
	products, err := client.GetProductsByCategory(ctx, "smartphones", ListOptions{Limit: 1})
	if err != nil || len(products.Products) != 1 || products.Products[0].Category != "smartphones" {
		t.Fatalf("GetProductsByCategory() = %+v, %v", products, err)
	}
	posts, err := client.GetPostsByUserID(ctx, 7, ListOptions{Limit: 2})
	if err != nil || len(posts.Posts) != 1 || posts.Posts[0].UserID != 7 {
		t.Fatalf("GetPostsByUserID() = %+v, %v", posts, err)
	}
	todos, err := client.GetTodos(ctx, ListOptions{Limit: 3})
	if err != nil || len(todos.Todos) != 1 || todos.Todos[0].ID != 2 {
		t.Fatalf("GetTodos() = %+v, %v", todos, err)
	}
	todo, err := client.GetTodoByID(ctx, 2)
	if err != nil || todo.UserID != 7 {
		t.Fatalf("GetTodoByID() = %+v, %v", todo, err)
	}
	userTodos, err := client.GetTodosByUserID(ctx, 7, ListOptions{Limit: 4})
	if err != nil || len(userTodos.Todos) != 1 || userTodos.Limit != 4 {
		t.Fatalf("GetTodosByUserID() = %+v, %v", userTodos, err)
	}
	recipes, err := client.GetRecipes(ctx, ListOptions{Limit: 2})
	if err != nil || len(recipes.Recipes) != 1 || recipes.Recipes[0].Name != "Pizza" {
		t.Fatalf("GetRecipes() = %+v, %v", recipes, err)
	}
	recipe, err := client.GetRecipeByID(ctx, 1)
	if err != nil || len(recipe.Ingredients) != 1 || recipe.MealType[0] != "Dinner" {
		t.Fatalf("GetRecipeByID() = %+v, %v", recipe, err)
	}
	searchRecipes, err := client.SearchRecipes(ctx, "pizza", ListOptions{Limit: 2})
	if err != nil || len(searchRecipes.Recipes) != 1 || searchRecipes.Recipes[0].Name != "Pizza" {
		t.Fatalf("SearchRecipes() = %+v, %v", searchRecipes, err)
	}
	userComments, err := client.GetCommentsByUserID(ctx, 7, ListOptions{Limit: 1, SortBy: "likes", Order: "desc"})
	if err != nil || len(userComments.Comments) != 1 || userComments.Comments[0].ID != 3 || userComments.Total != 2 {
		t.Fatalf("GetCommentsByUserID() = %+v, %v", userComments, err)
	}
}
