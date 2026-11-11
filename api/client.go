package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"dummyjson-explorer/models"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type ListOptions struct {
	Limit  int
	Skip   int
	SortBy string
	Order  string
	Select []string
}

func NewClient() *Client {
	return &Client{
		BaseURL: "https://dummyjson.com",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Reusable function for making GET requests and decoding JSON.
func (c *Client) getJSON(ctx context.Context, endpoint string, destination any) error {
	requestURL := c.BaseURL + endpoint

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("create DummyJSON request for %s: %w", endpoint, err)
	}

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("DummyJSON GET %s: %w", endpoint, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("DummyJSON GET %s returned %s (could not read error body: %w)", endpoint, response.Status, readErr)
		}
		message := strings.TrimSpace(string(body))
		if message == "" {
			return fmt.Errorf("DummyJSON GET %s returned %s", endpoint, response.Status)
		}
		return fmt.Errorf("DummyJSON GET %s returned %s: %s", endpoint, response.Status, message)
	}

	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		return fmt.Errorf("decode DummyJSON response for %s: %w", endpoint, err)
	}

	return nil
}

func listEndpoint(endpoint string, options ListOptions) (string, error) {
	if options.Limit < 0 {
		return "", fmt.Errorf("limit must not be negative")
	}
	if options.Skip < 0 {
		return "", fmt.Errorf("skip must not be negative")
	}
	if options.SortBy != "" && !isFieldName(options.SortBy) {
		return "", fmt.Errorf("sortBy must be a field name")
	}
	order := strings.ToLower(options.Order)
	if order != "" && order != "asc" && order != "desc" {
		return "", fmt.Errorf("order must be \"asc\" or \"desc\"")
	}
	if order != "" && options.SortBy == "" {
		return "", fmt.Errorf("sortBy is required when order is specified")
	}

	selected := make([]string, 0, len(options.Select))
	for _, field := range options.Select {
		field = strings.TrimSpace(field)
		if !isFieldName(field) {
			return "", fmt.Errorf("select fields must be non-empty field names")
		}
		selected = append(selected, field)
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse DummyJSON endpoint: %w", err)
	}
	query := parsed.Query()
	if options.Limit > 0 {
		query.Set("limit", fmt.Sprint(options.Limit))
	}
	if options.Skip > 0 {
		query.Set("skip", fmt.Sprint(options.Skip))
	}
	if options.SortBy != "" {
		query.Set("sortBy", options.SortBy)
	}
	if order != "" {
		query.Set("order", order)
	}
	if len(selected) > 0 {
		query.Set("select", strings.Join(selected, ","))
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func isFieldName(field string) bool {
	if field == "" || (field[0] < 'A' || field[0] > 'Z') && (field[0] < 'a' || field[0] > 'z') {
		return false
	}
	for i := 1; i < len(field); i++ {
		char := field[i]
		if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func applyListOptions(endpoint string, options ListOptions) (string, error) {
	if options.Limit == 0 && options.Skip == 0 && options.SortBy == "" && options.Order == "" && len(options.Select) == 0 {
		return endpoint, nil
	}
	return listEndpoint(endpoint, options)
}

// -------------------- PRODUCTS --------------------

// Fetch all products.
func (c *Client) GetProducts(ctx context.Context, options ListOptions) (*models.ProductsResponse, error) {
	var result models.ProductsResponse

	endpoint, err := applyListOptions("/products", options)
	if err != nil {
		return nil, err
	}
	err = c.getJSON(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Fetch one product using its ID.
func (c *Client) GetProductByID(ctx context.Context, id int) (*models.Product, error) {
	var product models.Product

	endpoint := fmt.Sprintf("/products/%d", id)

	err := c.getJSON(ctx, endpoint, &product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

// Search products using a keyword.
func (c *Client) SearchProducts(ctx context.Context, searchTerm string, options ListOptions) (*models.ProductsResponse, error) {
	if strings.TrimSpace(searchTerm) == "" {
		return nil, fmt.Errorf("search term must not be empty")
	}
	query := url.Values{}
	query.Set("q", searchTerm)

	endpoint, err := applyListOptions("/products/search?"+query.Encode(), options)
	if err != nil {
		return nil, err
	}

	var result models.ProductsResponse

	err = c.getJSON(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Fetch products belonging to a category.
func (c *Client) GetProductsByCategory(ctx context.Context, category string, options ListOptions) (*models.ProductsResponse, error) {
	if strings.TrimSpace(category) == "" {
		return nil, fmt.Errorf("category must not be empty")
	}

	var result models.ProductsResponse
	endpoint, err := applyListOptions("/products/category/"+url.PathEscape(category), options)
	if err != nil {
		return nil, err
	}
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// -------------------- USERS --------------------

// Fetch all users.
func (c *Client) GetUsers(ctx context.Context, options ListOptions) (*models.UsersResponse, error) {
	var result models.UsersResponse

	endpoint, err := applyListOptions("/users?limit=10", options)
	if err != nil {
		return nil, err
	}
	err = c.getJSON(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Fetch one user using their ID.
func (c *Client) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	var user models.User

	endpoint := fmt.Sprintf("/users/%d", id)

	err := c.getJSON(ctx, endpoint, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Search users by a keyword.
func (c *Client) SearchUsers(ctx context.Context, searchTerm string, options ListOptions) (*models.UsersResponse, error) {
	if strings.TrimSpace(searchTerm) == "" {
		return nil, fmt.Errorf("search term must not be empty")
	}

	query := url.Values{}
	query.Set("q", searchTerm)
	endpoint, err := applyListOptions("/users/search?"+query.Encode(), options)
	if err != nil {
		return nil, err
	}

	var result models.UsersResponse
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// -------------------- POSTS --------------------

// Fetch all posts.
func (c *Client) GetPosts(ctx context.Context, options ListOptions) (*models.PostsResponse, error) {
	var result models.PostsResponse

	endpoint, err := applyListOptions("/posts?limit=10", options)
	if err != nil {
		return nil, err
	}
	err = c.getJSON(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Fetch one post using its ID.
func (c *Client) GetPostByID(ctx context.Context, id int) (*models.Post, error) {
	var post models.Post

	endpoint := fmt.Sprintf("/posts/%d", id)

	err := c.getJSON(ctx, endpoint, &post)
	if err != nil {
		return nil, err
	}

	return &post, nil
}

// Fetch posts belonging to a particular user.
func (c *Client) GetPostsByUserID(ctx context.Context, userID int, options ListOptions) (*models.PostsResponse, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user ID must be greater than zero")
	}

	var result models.PostsResponse
	endpoint, err := applyListOptions(fmt.Sprintf("/users/%d/posts", userID), options)
	if err != nil {
		return nil, err
	}
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Search posts using a keyword.
func (c *Client) SearchPosts(ctx context.Context, searchTerm string, options ListOptions) (*models.PostsResponse, error) {
	if strings.TrimSpace(searchTerm) == "" {
		return nil, fmt.Errorf("search term must not be empty")
	}
	query := url.Values{}
	query.Set("q", searchTerm)

	endpoint, err := applyListOptions("/posts/search?"+query.Encode(), options)
	if err != nil {
		return nil, err
	}

	var result models.PostsResponse

	err = c.getJSON(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// -------------------- TODOS --------------------

// Fetch todos.
func (c *Client) GetTodos(ctx context.Context, options ListOptions) (*models.TodosResponse, error) {
	var result models.TodosResponse
	endpoint, err := applyListOptions("/todos?limit=10", options)
	if err != nil {
		return nil, err
	}
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Fetch one todo using its ID.
func (c *Client) GetTodoByID(ctx context.Context, id int) (*models.Todo, error) {
	if id <= 0 {
		return nil, fmt.Errorf("todo ID must be greater than zero")
	}
	var todo models.Todo
	if err := c.getJSON(ctx, fmt.Sprintf("/todos/%d", id), &todo); err != nil {
		return nil, err
	}
	return &todo, nil
}

// Fetch todos belonging to a particular user.
func (c *Client) GetTodosByUserID(ctx context.Context, userID int, options ListOptions) (*models.TodosResponse, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user ID must be greater than zero")
	}

	var result models.TodosResponse
	endpoint, err := applyListOptions(fmt.Sprintf("/todos/user/%d", userID), options)
	if err != nil {
		return nil, err
	}
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// -------------------- RECIPES --------------------

// Fetch recipes.
func (c *Client) GetRecipes(ctx context.Context, options ListOptions) (*models.RecipesResponse, error) {
	var result models.RecipesResponse
	endpoint, err := applyListOptions("/recipes", options)
	if err != nil {
		return nil, err
	}
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Fetch one recipe using its ID.
func (c *Client) GetRecipeByID(ctx context.Context, id int) (*models.Recipe, error) {
	if id <= 0 {
		return nil, fmt.Errorf("recipe ID must be greater than zero")
	}
	var recipe models.Recipe
	if err := c.getJSON(ctx, fmt.Sprintf("/recipes/%d", id), &recipe); err != nil {
		return nil, err
	}
	return &recipe, nil
}

// Search recipes by a keyword.
func (c *Client) SearchRecipes(ctx context.Context, searchTerm string, options ListOptions) (*models.RecipesResponse, error) {
	if strings.TrimSpace(searchTerm) == "" {
		return nil, fmt.Errorf("search term must not be empty")
	}
	query := url.Values{}
	query.Set("q", searchTerm)
	endpoint, err := applyListOptions("/recipes/search?"+query.Encode(), options)
	if err != nil {
		return nil, err
	}
	var result models.RecipesResponse
	if err := c.getJSON(ctx, endpoint, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// -------------------- COMMENTS --------------------

// Fetch all comments.
func (c *Client) GetComments(ctx context.Context, options ListOptions) (*models.CommentsResponse, error) {
	var result models.CommentsResponse

	endpoint, err := applyListOptions("/comments?limit=10", options)
	if err != nil {
		return nil, err
	}
	err = c.getJSON(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Fetch comments belonging to a particular post.
func (c *Client) GetCommentsByPostID(ctx context.Context, postID int, options ListOptions) (*models.CommentsResponse, error) {
	var result models.CommentsResponse

	endpoint, err := applyListOptions(fmt.Sprintf("/comments/post/%d", postID), options)
	if err != nil {
		return nil, err
	}

	err = c.getJSON(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Fetch comments by their author. DummyJSON has no comments-by-user route, so
// this filters the complete comments collection before applying local options.
func (c *Client) GetCommentsByUserID(ctx context.Context, userID int, options ListOptions) (*models.CommentsResponse, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user ID must be greater than zero")
	}
	if len(options.Select) > 0 {
		return nil, fmt.Errorf("field selection is not supported for comments filtered by user")
	}
	if options.SortBy != "" && options.SortBy != "id" && options.SortBy != "postId" && options.SortBy != "likes" {
		return nil, fmt.Errorf("sortBy for comments filtered by user must be id, postId, or likes")
	}
	if options.Limit < 0 || options.Skip < 0 {
		return nil, fmt.Errorf("limit and skip must not be negative")
	}
	order := strings.ToLower(options.Order)
	if order != "" && order != "asc" && order != "desc" {
		return nil, fmt.Errorf("order must be \"asc\" or \"desc\"")
	}
	if order != "" && options.SortBy == "" {
		return nil, fmt.Errorf("sortBy is required when order is specified")
	}

	var allComments models.CommentsResponse
	if err := c.getJSON(ctx, "/comments?limit=0", &allComments); err != nil {
		return nil, err
	}
	filtered := make([]models.Comment, 0)
	for _, comment := range allComments.Comments {
		if comment.User.ID == userID {
			filtered = append(filtered, comment)
		}
	}

	if options.SortBy != "" {
		sort.SliceStable(filtered, func(i, j int) bool {
			var less bool
			switch options.SortBy {
			case "id":
				less = filtered[i].ID < filtered[j].ID
			case "postId":
				less = filtered[i].PostID < filtered[j].PostID
			case "likes":
				less = filtered[i].Likes < filtered[j].Likes
			}
			if order == "desc" {
				return !less && commentSortValuesDiffer(filtered[i], filtered[j], options.SortBy)
			}
			return less
		})
	}

	total := len(filtered)
	skip := min(options.Skip, total)
	filtered = filtered[skip:]
	limit := options.Limit
	if limit == 0 {
		limit = 10
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return &models.CommentsResponse{
		Comments: filtered,
		Total:    total,
		Skip:     skip,
		Limit:    limit,
	}, nil
}

func commentSortValuesDiffer(a, b models.Comment, field string) bool {
	switch field {
	case "id":
		return a.ID != b.ID
	case "postId":
		return a.PostID != b.PostID
	case "likes":
		return a.Likes != b.Likes
	default:
		return false
	}
}

// -------------------- CARTS --------------------

// Fetch all carts.
func (c *Client) GetCarts(ctx context.Context, options ListOptions) (*models.CartsResponse, error) {
	var result models.CartsResponse

	endpoint, err := applyListOptions("/carts?limit=10", options)
	if err != nil {
		return nil, err
	}
	err = c.getJSON(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Fetch one cart using its ID.
func (c *Client) GetCartByID(ctx context.Context, id int) (*models.Cart, error) {
	var cart models.Cart

	endpoint := fmt.Sprintf("/carts/%d", id)

	err := c.getJSON(ctx, endpoint, &cart)
	if err != nil {
		return nil, err
	}

	return &cart, nil
}

// Fetch carts belonging to a particular user.
func (c *Client) GetCartsByUserID(ctx context.Context, userID int, options ListOptions) (*models.CartsResponse, error) {
	var result models.CartsResponse

	endpoint, err := applyListOptions(fmt.Sprintf("/carts/user/%d", userID), options)
	if err != nil {
		return nil, err
	}

	err = c.getJSON(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
