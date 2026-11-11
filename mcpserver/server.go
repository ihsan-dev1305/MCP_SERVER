package mcpserver

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"dummyjson-explorer/api"
	"dummyjson-explorer/models"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
)

type GetUserInput struct {
	ID int `json:"id" jsonschema:"the ID of the user to retrieve"`
}

type GetUserOverviewInput struct {
	UserID int `json:"userId" jsonschema:"the ID of the user whose overview to retrieve"`
}

type UserOverview struct {
	User  *models.User  `json:"user"`
	Posts []models.Post `json:"posts"`
	Todos []models.Todo `json:"todos"`
	Carts []models.Cart `json:"carts"`
}

type GetShoppingSummaryInput struct {
	UserID int `json:"userId" jsonschema:"the ID of the user whose shopping summary to retrieve"`
}

type ShoppingSummary struct {
	UserID          int           `json:"userId"`
	Carts           []models.Cart `json:"carts"`
	CartCount       int           `json:"cartCount"`
	TotalProducts   int           `json:"totalProducts"`
	TotalQuantity   int           `json:"totalQuantity"`
	TotalValue      float64       `json:"totalValue"`
	DiscountedValue float64       `json:"discountedValue"`
}

type GetPostDetailsInput struct {
	PostID int `json:"postId" jsonschema:"the ID of the post whose details to retrieve"`
}

type PostDetails struct {
	Post     *models.Post     `json:"post"`
	Comments []models.Comment `json:"comments"`
	Author   *models.User     `json:"author"`
}

type GetProductInput struct {
	ID int `json:"id" jsonschema:"the ID of the product to retrieve"`
}

type GetProductsByCategoryInput struct {
	Category string `json:"category" jsonschema:"the product category to retrieve"`
	ListInput
}

type CompareProductsInput struct {
	ProductIDs []int `json:"productIds" jsonschema:"one or more product IDs to compare"`
}

type ProductComparison struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Rating    float64 `json:"rating"`
	Category  string  `json:"category"`
	Available bool    `json:"available"`
	Stock     int     `json:"stock"`
}

type CompareProductsResult struct {
	Products []ProductComparison `json:"products"`
}

type SearchProductsInput struct {
	SearchTerm string `json:"searchTerm" jsonschema:"the keyword to search for"`
	ListInput
}

type SearchUsersInput struct {
	SearchTerm string `json:"searchTerm" jsonschema:"the keyword to search for"`
	ListInput
}

type ListInput struct {
	Limit  int      `json:"limit,omitempty" jsonschema:"maximum number of records to return; defaults to the API's existing page size"`
	Skip   int      `json:"skip,omitempty" jsonschema:"number of records to skip"`
	SortBy string   `json:"sortBy,omitempty" jsonschema:"field to sort by"`
	Order  string   `json:"order,omitempty" jsonschema:"sort direction: asc or desc (requires sortBy)"`
	Select []string `json:"select,omitempty" jsonschema:"fields to include in each record"`
}

type GetPostInput struct {
	ID int `json:"id" jsonschema:"the ID of the post to retrieve"`
}

type SearchPostsInput struct {
	SearchTerm string `json:"searchTerm" jsonschema:"the keyword to search for"`
	ListInput
}

type GetPostsByUserInput struct {
	UserID int `json:"userId" jsonschema:"the ID of the user whose posts to retrieve"`
	ListInput
}

type GetCommentsByPostInput struct {
	PostID int `json:"postId" jsonschema:"the ID of the post whose comments to retrieve"`
	ListInput
}

type GetCartInput struct {
	ID int `json:"id" jsonschema:"the ID of the cart to retrieve"`
}

type GetCartsByUserInput struct {
	UserID int `json:"userId" jsonschema:"the ID of the user whose carts to retrieve"`
	ListInput
}

type GetTodosByUserInput struct {
	UserID int `json:"userId" jsonschema:"the ID of the user whose todos to retrieve"`
	ListInput
}

type GetCommentsByUserInput struct {
	UserID int    `json:"userId" jsonschema:"the ID of the comment author"`
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum number of matching comments to return; defaults to 10"`
	Skip   int    `json:"skip,omitempty" jsonschema:"number of matching comments to skip"`
	SortBy string `json:"sortBy,omitempty" jsonschema:"one of id, postId, or likes"`
	Order  string `json:"order,omitempty" jsonschema:"sort direction: asc or desc (requires sortBy)"`
}

type GetTodoInput struct {
	ID int `json:"id" jsonschema:"the ID of the todo to retrieve"`
}

type GetRecipeInput struct {
	ID int `json:"id" jsonschema:"the ID of the recipe to retrieve"`
}

type SearchRecipesInput struct {
	SearchTerm string `json:"searchTerm" jsonschema:"the keyword to search for"`
	ListInput
}

func (input ListInput) options() api.ListOptions {
	return api.ListOptions{
		Limit:  input.Limit,
		Skip:   input.Skip,
		SortBy: input.SortBy,
		Order:  input.Order,
		Select: input.Select,
	}
}

func (input GetCommentsByUserInput) options() api.ListOptions {
	return api.ListOptions{
		Limit:  input.Limit,
		Skip:   input.Skip,
		SortBy: input.SortBy,
		Order:  input.Order,
	}
}

func getUserOverview(ctx context.Context, client *api.Client, userID int) (*UserOverview, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("userId must be greater than zero")
	}

	group, groupCtx := errgroup.WithContext(ctx)
	overview := &UserOverview{}

	group.Go(func() error {
		user, err := client.GetUserByID(groupCtx, userID)
		if err != nil {
			return fmt.Errorf("fetch user: %w", err)
		}
		overview.User = user
		return nil
	})
	group.Go(func() error {
		posts, err := client.GetPostsByUserID(groupCtx, userID, api.ListOptions{})
		if err != nil {
			return fmt.Errorf("fetch user posts: %w", err)
		}
		overview.Posts = posts.Posts
		return nil
	})
	group.Go(func() error {
		todos, err := client.GetTodosByUserID(groupCtx, userID, api.ListOptions{})
		if err != nil {
			return fmt.Errorf("fetch user todos: %w", err)
		}
		overview.Todos = todos.Todos
		return nil
	})
	group.Go(func() error {
		carts, err := client.GetCartsByUserID(groupCtx, userID, api.ListOptions{})
		if err != nil {
			return fmt.Errorf("fetch user carts: %w", err)
		}
		overview.Carts = carts.Carts
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("get user overview: %w", err)
	}
	if overview.Posts == nil {
		overview.Posts = []models.Post{}
	}
	if overview.Todos == nil {
		overview.Todos = []models.Todo{}
	}
	if overview.Carts == nil {
		overview.Carts = []models.Cart{}
	}
	return overview, nil
}

func getPostDetails(ctx context.Context, client *api.Client, postID int) (*PostDetails, error) {
	if postID <= 0 {
		return nil, fmt.Errorf("postId must be greater than zero")
	}

	post, err := client.GetPostByID(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("fetch post: %w", err)
	}

	group, groupCtx := errgroup.WithContext(ctx)
	details := &PostDetails{Post: post}
	group.Go(func() error {
		comments, err := client.GetCommentsByPostID(groupCtx, postID, api.ListOptions{})
		if err != nil {
			return fmt.Errorf("fetch post comments: %w", err)
		}
		details.Comments = comments.Comments
		return nil
	})
	group.Go(func() error {
		author, err := client.GetUserByID(groupCtx, post.UserID)
		if err != nil {
			return fmt.Errorf("fetch post author: %w", err)
		}
		details.Author = author
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("get post details: %w", err)
	}
	if details.Comments == nil {
		details.Comments = []models.Comment{}
	}
	return details, nil
}

func compareProducts(ctx context.Context, client *api.Client, productIDs []int) (*CompareProductsResult, error) {
	if len(productIDs) == 0 {
		return nil, fmt.Errorf("productIds must contain at least one product ID")
	}

	type lookup struct {
		id      int
		indexes []int
	}
	lookups := make([]lookup, 0, len(productIDs))
	lookupByID := make(map[int]int, len(productIDs))
	for index, id := range productIDs {
		if id <= 0 {
			return nil, fmt.Errorf("productIds[%d] must be greater than zero", index)
		}
		if lookupIndex, ok := lookupByID[id]; ok {
			lookups[lookupIndex].indexes = append(lookups[lookupIndex].indexes, index)
			continue
		}
		lookupByID[id] = len(lookups)
		lookups = append(lookups, lookup{id: id, indexes: []int{index}})
	}

	result := &CompareProductsResult{
		Products: make([]ProductComparison, len(productIDs)),
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(5)
	for _, productLookup := range lookups {
		productLookup := productLookup
		group.Go(func() error {
			product, err := client.GetProductByID(groupCtx, productLookup.id)
			if err != nil {
				return fmt.Errorf("fetch product %d: %w", productLookup.id, err)
			}

			comparison := ProductComparison{
				ID:        product.ID,
				Name:      product.Title,
				Price:     product.Price,
				Rating:    product.Rating,
				Category:  product.Category,
				Available: product.Stock > 0,
				Stock:     product.Stock,
			}
			for _, index := range productLookup.indexes {
				result.Products[index] = comparison
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("compare products: %w", err)
	}
	return result, nil
}

func getShoppingSummary(ctx context.Context, client *api.Client, userID int) (*ShoppingSummary, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("userId must be greater than zero")
	}

	cartsResponse, err := client.GetCartsByUserID(ctx, userID, api.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("fetch user carts: %w", err)
	}
	carts := cartsResponse.Carts
	if carts == nil {
		carts = []models.Cart{}
	}

	summary := &ShoppingSummary{
		UserID:    userID,
		Carts:     carts,
		CartCount: len(carts),
	}
	for _, cart := range carts {
		summary.TotalProducts += len(cart.Products)
		summary.TotalValue += cart.Total
		summary.DiscountedValue += cart.DiscountedTotal
		for _, product := range cart.Products {
			summary.TotalQuantity += product.Quantity
		}
	}
	return summary, nil
}

func addPortableTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	if tool.InputSchema == nil {
		inputSchema, err := jsonschema.For[In](nil)
		if err != nil {
			panic(fmt.Sprintf("infer input schema for tool %q: %v", tool.Name, err))
		}
		normalizeNullableTypes(inputSchema, make(map[*jsonschema.Schema]bool))
		tool.InputSchema = inputSchema
	}

	if tool.OutputSchema == nil && reflect.TypeFor[Out]() != reflect.TypeFor[any]() {
		outputType := reflect.TypeFor[Out]()
		if outputType.Kind() == reflect.Pointer {
			outputType = outputType.Elem()
		}
		outputSchema, err := jsonschema.ForType(outputType, &jsonschema.ForOptions{})
		if err != nil {
			panic(fmt.Sprintf("infer output schema for tool %q: %v", tool.Name, err))
		}
		normalizeNullableTypes(outputSchema, make(map[*jsonschema.Schema]bool))
		tool.OutputSchema = outputSchema
	}

	mcp.AddTool(server, tool, handler)
}

func normalizeNullableTypes(schema *jsonschema.Schema, visited map[*jsonschema.Schema]bool) {
	if schema == nil || visited[schema] {
		return
	}
	visited[schema] = true

	if len(schema.Types) > 1 {
		branches := make([]*jsonschema.Schema, 0, len(schema.Types))
		for _, schemaType := range schema.Types {
			branches = append(branches, &jsonschema.Schema{Type: schemaType})
		}
		schema.Types = nil
		schema.AnyOf = append(branches, schema.AnyOf...)
	} else if len(schema.Types) == 1 {
		schema.Type = schema.Types[0]
		schema.Types = nil
	}

	for _, child := range schema.Properties {
		normalizeNullableTypes(child, visited)
	}
	for _, child := range schema.PatternProperties {
		normalizeNullableTypes(child, visited)
	}
	for _, child := range schema.Defs {
		normalizeNullableTypes(child, visited)
	}
	for _, child := range schema.Definitions {
		normalizeNullableTypes(child, visited)
	}
	for _, child := range schema.DependentSchemas {
		normalizeNullableTypes(child, visited)
	}
	for _, child := range schema.AllOf {
		normalizeNullableTypes(child, visited)
	}
	for _, child := range schema.AnyOf {
		normalizeNullableTypes(child, visited)
	}
	for _, child := range schema.OneOf {
		normalizeNullableTypes(child, visited)
	}
	for _, child := range []*jsonschema.Schema{
		schema.Items,
		schema.AdditionalItems,
		schema.Contains,
		schema.UnevaluatedItems,
		schema.AdditionalProperties,
		schema.PropertyNames,
		schema.UnevaluatedProperties,
		schema.Not,
		schema.If,
		schema.Then,
		schema.Else,
		schema.ContentSchema,
	} {
		normalizeNullableTypes(child, visited)
	}
	for _, children := range [][]*jsonschema.Schema{schema.ItemsArray, schema.PrefixItems} {
		for _, child := range children {
			normalizeNullableTypes(child, visited)
		}
	}
}

func NewServer(client *api.Client) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "dummyjson-explorer",
			Version: "1.0.0",
		},
		nil,
	)

	// Tool 1: List products.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "list_products",
			Description: "Fetch products from the DummyJSON API",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input ListInput,
		) (*mcp.CallToolResult, *models.ProductsResponse, error) {

			products, err := client.GetProducts(ctx, input.options())
			if err != nil {
				return nil, nil, err
			}

			return nil, products, nil
		},
	)

	// Tool 2: Get one product by ID.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_product",
			Description: "Fetch a single product by its ID from DummyJSON",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetProductInput,
		) (*mcp.CallToolResult, *models.Product, error) {

			if input.ID <= 0 {
				return nil, nil, fmt.Errorf("id must be greater than zero")
			}
			product, err := client.GetProductByID(ctx, input.ID)
			if err != nil {
				return nil, nil, err
			}

			return nil, product, nil
		},
	)

	// Combined workflow: fetch multiple products for side-by-side comparison.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "compare_products",
			Description: "Compare products by name, price, rating, category, availability, and stock",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input CompareProductsInput,
		) (*mcp.CallToolResult, *CompareProductsResult, error) {
			result, err := compareProducts(ctx, client, input.ProductIDs)
			if err != nil {
				return nil, nil, err
			}
			return nil, result, nil
		},
	)

	// Tool 3: Search products.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "search_products",
			Description: "Search DummyJSON products using a keyword",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input SearchProductsInput,
		) (*mcp.CallToolResult, *models.ProductsResponse, error) {

			if strings.TrimSpace(input.SearchTerm) == "" {
				return nil, nil, fmt.Errorf("searchTerm must not be empty")
			}
			products, err := client.SearchProducts(ctx, input.SearchTerm, input.ListInput.options())
			if err != nil {
				return nil, nil, err
			}

			return nil, products, nil
		},
	)

	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_products_by_category",
			Description: "Fetch products in a DummyJSON category",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetProductsByCategoryInput,
		) (*mcp.CallToolResult, *models.ProductsResponse, error) {
			if strings.TrimSpace(input.Category) == "" {
				return nil, nil, fmt.Errorf("category must not be empty")
			}
			products, err := client.GetProductsByCategory(ctx, input.Category, input.ListInput.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, products, nil
		},
	)

	// Tool 4: List users.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "list_users",
			Description: "Fetch users from the DummyJSON API",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input ListInput,
		) (*mcp.CallToolResult, *models.UsersResponse, error) {

			users, err := client.GetUsers(ctx, input.options())
			if err != nil {
				return nil, nil, err
			}

			return nil, users, nil
		},
	)

	// Tool 5: Get one user by ID.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_user",
			Description: "Fetch a single user by their ID",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetUserInput,
		) (*mcp.CallToolResult, *models.User, error) {

			if input.ID <= 0 {
				return nil, nil, fmt.Errorf("id must be greater than zero")
			}
			user, err := client.GetUserByID(ctx, input.ID)
			if err != nil {
				return nil, nil, err
			}

			return nil, user, nil
		},
	)

	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "search_users",
			Description: "Search DummyJSON users by a keyword",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input SearchUsersInput,
		) (*mcp.CallToolResult, *models.UsersResponse, error) {
			if strings.TrimSpace(input.SearchTerm) == "" {
				return nil, nil, fmt.Errorf("searchTerm must not be empty")
			}
			users, err := client.SearchUsers(ctx, input.SearchTerm, input.ListInput.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, users, nil
		},
	)

	// Combined workflow: fetch a user's profile, posts, todos, and carts.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_user_overview",
			Description: "Fetch a user's profile, posts, todos, and carts",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetUserOverviewInput,
		) (*mcp.CallToolResult, *UserOverview, error) {
			overview, err := getUserOverview(ctx, client, input.UserID)
			if err != nil {
				return nil, nil, err
			}
			return nil, overview, nil
		},
	)

	// Tool 6: List posts.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "list_posts",
			Description: "Fetch posts from the DummyJSON API",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input ListInput,
		) (*mcp.CallToolResult, *models.PostsResponse, error) {
			posts, err := client.GetPosts(ctx, input.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, posts, nil
		},
	)

	// Tool 7: Get one post by ID.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_post",
			Description: "Fetch a single post by its ID from DummyJSON",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetPostInput,
		) (*mcp.CallToolResult, *models.Post, error) {
			if input.ID <= 0 {
				return nil, nil, fmt.Errorf("id must be greater than zero")
			}
			post, err := client.GetPostByID(ctx, input.ID)
			if err != nil {
				return nil, nil, err
			}
			return nil, post, nil
		},
	)

	// Combined workflow: fetch a post, its comments, and its author.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_post_details",
			Description: "Fetch a post with its comments and author information",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetPostDetailsInput,
		) (*mcp.CallToolResult, *PostDetails, error) {
			details, err := getPostDetails(ctx, client, input.PostID)
			if err != nil {
				return nil, nil, err
			}
			return nil, details, nil
		},
	)

	// Tool 8: Search posts.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "search_posts",
			Description: "Search DummyJSON posts using a keyword",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input SearchPostsInput,
		) (*mcp.CallToolResult, *models.PostsResponse, error) {
			if strings.TrimSpace(input.SearchTerm) == "" {
				return nil, nil, fmt.Errorf("searchTerm must not be empty")
			}
			posts, err := client.SearchPosts(ctx, input.SearchTerm, input.ListInput.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, posts, nil
		},
	)

	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_posts_by_user",
			Description: "Fetch posts written by a user",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetPostsByUserInput,
		) (*mcp.CallToolResult, *models.PostsResponse, error) {
			if input.UserID <= 0 {
				return nil, nil, fmt.Errorf("userId must be greater than zero")
			}
			posts, err := client.GetPostsByUserID(ctx, input.UserID, input.ListInput.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, posts, nil
		},
	)

	// Tool 9: List comments.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "list_comments",
			Description: "Fetch comments from the DummyJSON API",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input ListInput,
		) (*mcp.CallToolResult, *models.CommentsResponse, error) {
			comments, err := client.GetComments(ctx, input.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, comments, nil
		},
	)

	// Tool 10: Get comments for one post.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_comments_by_post",
			Description: "Fetch comments belonging to a post by its ID",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetCommentsByPostInput,
		) (*mcp.CallToolResult, *models.CommentsResponse, error) {
			if input.PostID <= 0 {
				return nil, nil, fmt.Errorf("postId must be greater than zero")
			}
			comments, err := client.GetCommentsByPostID(ctx, input.PostID, input.ListInput.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, comments, nil
		},
	)

	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_comments_by_user",
			Description: "Fetch comments written by a user; DummyJSON has no direct user-comments endpoint",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetCommentsByUserInput,
		) (*mcp.CallToolResult, *models.CommentsResponse, error) {
			comments, err := client.GetCommentsByUserID(ctx, input.UserID, input.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, comments, nil
		},
	)

	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "list_todos",
			Description: "Fetch todos from the DummyJSON API",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input ListInput,
		) (*mcp.CallToolResult, *models.TodosResponse, error) {
			todos, err := client.GetTodos(ctx, input.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, todos, nil
		},
	)

	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_todo",
			Description: "Fetch a single todo by its ID",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetTodoInput,
		) (*mcp.CallToolResult, *models.Todo, error) {
			if input.ID <= 0 {
				return nil, nil, fmt.Errorf("id must be greater than zero")
			}
			todo, err := client.GetTodoByID(ctx, input.ID)
			if err != nil {
				return nil, nil, err
			}
			return nil, todo, nil
		},
	)

	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_todos_by_user",
			Description: "Fetch todos belonging to a user",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetTodosByUserInput,
		) (*mcp.CallToolResult, *models.TodosResponse, error) {
			if input.UserID <= 0 {
				return nil, nil, fmt.Errorf("userId must be greater than zero")
			}
			todos, err := client.GetTodosByUserID(ctx, input.UserID, input.ListInput.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, todos, nil
		},
	)

	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "list_recipes",
			Description: "Fetch recipes from the DummyJSON API",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input ListInput,
		) (*mcp.CallToolResult, *models.RecipesResponse, error) {
			recipes, err := client.GetRecipes(ctx, input.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, recipes, nil
		},
	)

	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_recipe",
			Description: "Fetch a single recipe by its ID",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetRecipeInput,
		) (*mcp.CallToolResult, *models.Recipe, error) {
			if input.ID <= 0 {
				return nil, nil, fmt.Errorf("id must be greater than zero")
			}
			recipe, err := client.GetRecipeByID(ctx, input.ID)
			if err != nil {
				return nil, nil, err
			}
			return nil, recipe, nil
		},
	)

	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "search_recipes",
			Description: "Search DummyJSON recipes by a keyword",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input SearchRecipesInput,
		) (*mcp.CallToolResult, *models.RecipesResponse, error) {
			if strings.TrimSpace(input.SearchTerm) == "" {
				return nil, nil, fmt.Errorf("searchTerm must not be empty")
			}
			recipes, err := client.SearchRecipes(ctx, input.SearchTerm, input.ListInput.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, recipes, nil
		},
	)

	// Tool 11: List carts.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "list_carts",
			Description: "Fetch carts from the DummyJSON API",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input ListInput,
		) (*mcp.CallToolResult, *models.CartsResponse, error) {
			carts, err := client.GetCarts(ctx, input.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, carts, nil
		},
	)

	// Tool 12: Get one cart by ID.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_cart",
			Description: "Fetch a single cart by its ID from DummyJSON",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetCartInput,
		) (*mcp.CallToolResult, *models.Cart, error) {
			if input.ID <= 0 {
				return nil, nil, fmt.Errorf("id must be greater than zero")
			}
			cart, err := client.GetCartByID(ctx, input.ID)
			if err != nil {
				return nil, nil, err
			}
			return nil, cart, nil
		},
	)

	// Tool 13: Get carts for one user.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_carts_by_user",
			Description: "Fetch carts belonging to a user by their ID",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetCartsByUserInput,
		) (*mcp.CallToolResult, *models.CartsResponse, error) {
			if input.UserID <= 0 {
				return nil, nil, fmt.Errorf("userId must be greater than zero")
			}
			carts, err := client.GetCartsByUserID(ctx, input.UserID, input.ListInput.options())
			if err != nil {
				return nil, nil, err
			}
			return nil, carts, nil
		},
	)

	// Combined workflow: summarize a user's carts using the cart endpoint data.
	addPortableTool(
		server,
		&mcp.Tool{
			Name:        "get_shopping_summary",
			Description: "Summarize a user's carts, products, quantities, and totals",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input GetShoppingSummaryInput,
		) (*mcp.CallToolResult, *ShoppingSummary, error) {
			summary, err := getShoppingSummary(ctx, client, input.UserID)
			if err != nil {
				return nil, nil, err
			}
			return nil, summary, nil
		},
	)

	return server
}
