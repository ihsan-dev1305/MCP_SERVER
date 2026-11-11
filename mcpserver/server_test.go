package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"dummyjson-explorer/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewServerExposesAllAPIEndpoints(t *testing.T) {
	ctx := context.Background()
	server := NewServer(api.NewClient())
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{Name: "server-test", Version: "1.0.0"},
		nil,
	)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]bool{
		"list_products":            true,
		"get_product":              true,
		"get_products_by_category": true,
		"compare_products":         true,
		"search_products":          true,
		"list_users":               true,
		"get_user":                 true,
		"search_users":             true,
		"get_user_overview":        true,
		"list_posts":               true,
		"get_post":                 true,
		"get_post_details":         true,
		"search_posts":             true,
		"get_posts_by_user":        true,
		"list_comments":            true,
		"get_comments_by_post":     true,
		"get_comments_by_user":     true,
		"list_todos":               true,
		"get_todo":                 true,
		"get_todos_by_user":        true,
		"list_recipes":             true,
		"get_recipe":               true,
		"search_recipes":           true,
		"list_carts":               true,
		"get_cart":                 true,
		"get_carts_by_user":        true,
		"get_shopping_summary":     true,
	}
	for _, tool := range result.Tools {
		if !expected[tool.Name] {
			t.Errorf("unexpected tool %q", tool.Name)
		}
		delete(expected, tool.Name)
	}
	if len(expected) != 0 {
		t.Errorf("missing tools: %v", expected)
	}

	for _, tool := range result.Tools {
		assertNoUnionTypes(t, tool.Name+".inputSchema", tool.InputSchema)
		assertNoUnionTypes(t, tool.Name+".outputSchema", tool.OutputSchema)
	}

	for _, toolName := range []string{"list_products", "list_users", "list_posts", "list_comments", "list_carts", "list_todos", "list_recipes"} {
		var found bool
		for _, tool := range result.Tools {
			if tool.Name != toolName {
				continue
			}
			found = true
			schema, ok := tool.InputSchema.(map[string]any)
			if !ok {
				t.Fatalf("%s input schema has type %T, want map[string]any", toolName, tool.InputSchema)
			}
			properties, ok := schema["properties"].(map[string]any)
			if !ok {
				t.Fatalf("%s input schema properties have type %T, want map[string]any", toolName, schema["properties"])
			}
			for _, property := range []string{"limit", "skip", "sortBy", "order", "select"} {
				if _, ok := properties[property]; !ok {
					t.Errorf("%s input schema is missing %q", toolName, property)
				}
			}
		}
		if !found {
			t.Errorf("tool %q was not returned by tools/list", toolName)
		}
	}

	for _, tool := range result.Tools {
		if tool.Name != "compare_products" {
			continue
		}
		schema, ok := tool.InputSchema.(map[string]any)
		if !ok {
			t.Fatalf("compare_products input schema has type %T, want map[string]any", tool.InputSchema)
		}
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("compare_products input schema properties have type %T, want map[string]any", schema["properties"])
		}
		productIDs, ok := properties["productIds"].(map[string]any)
		if !ok {
			t.Fatalf("compare_products productIds schema has type %T, want map[string]any", properties["productIds"])
		}
		branches, ok := productIDs["anyOf"].([]any)
		if !ok || len(branches) != 2 {
			t.Fatalf("compare_products productIds anyOf = %v, want null and array branches", productIDs["anyOf"])
		}
		var arrayBranch map[string]any
		hasNullBranch := false
		for _, branch := range branches {
			branchSchema, ok := branch.(map[string]any)
			if !ok {
				t.Fatalf("compare_products productIds branch has type %T, want map[string]any", branch)
			}
			switch branchSchema["type"] {
			case "null":
				hasNullBranch = true
			case "array":
				arrayBranch = branchSchema
			}
		}
		if !hasNullBranch || arrayBranch == nil {
			t.Errorf("compare_products productIds branches = %v, want null and array", branches)
			return
		}
		items, ok := productIDs["items"].(map[string]any)
		if !ok || items["type"] != "integer" {
			t.Errorf("compare_products productIds items schema = %v, want integer items", productIDs["items"])
		}
		return
	}
	t.Fatal("compare_products tool was not returned by tools/list")
}

func assertNoUnionTypes(t *testing.T, path string, value any) {
	t.Helper()

	switch schema := value.(type) {
	case map[string]any:
		if _, ok := schema["type"].([]any); ok {
			t.Errorf("%s contains a non-portable type array: %v", path, schema["type"])
		}
		for key, child := range schema {
			assertNoUnionTypes(t, path+"."+key, child)
		}
	case []any:
		for index, child := range schema {
			assertNoUnionTypes(t, fmt.Sprintf("%s[%d]", path, index), child)
		}
	}
}

func TestGetUserOverviewToolFetchesRelatedDataConcurrently(t *testing.T) {
	var requestCount atomic.Int32
	allRequestsStarted := make(chan struct{})
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount.Add(1) == 4 {
			close(allRequestsStarted)
		}
		select {
		case <-allRequestsStarted:
		case <-time.After(3 * time.Second):
			http.Error(w, "related requests were not concurrent", http.StatusGatewayTimeout)
			return
		}

		switch r.URL.Path {
		case "/users/1":
			fmt.Fprint(w, `{"id":1,"firstName":"Alex","lastName":"Doe","username":"alex"}`)
		case "/users/1/posts":
			fmt.Fprint(w, `{"posts":[{"id":10,"title":"A post","userId":1}],"total":1,"skip":0,"limit":1}`)
		case "/todos/user/1":
			fmt.Fprint(w, `{"todos":[{"id":20,"todo":"Learn Go","completed":false,"userId":1}],"total":1,"skip":0,"limit":1}`)
		case "/carts/user/1":
			fmt.Fprint(w, `{"carts":[{"id":30,"products":[],"userId":1}],"total":1,"skip":0,"limit":1}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer apiServer.Close()

	apiClient := &api.Client{BaseURL: apiServer.URL, HTTPClient: apiServer.Client()}
	server := NewServer(apiClient)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{Name: "overview-test", Version: "1.0.0"},
		nil,
	)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_user_overview",
		Arguments: map[string]any{"userId": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("get_user_overview returned tool error: %v", result.GetError())
	}

	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var overview UserOverview
	if err := json.Unmarshal(encoded, &overview); err != nil {
		t.Fatal(err)
	}
	if overview.User == nil || overview.User.ID != 1 {
		t.Errorf("overview user = %+v, want user ID 1", overview.User)
	}
	if len(overview.Posts) != 1 || overview.Posts[0].ID != 10 {
		t.Errorf("overview posts = %+v, want post ID 10", overview.Posts)
	}
	if len(overview.Todos) != 1 || overview.Todos[0].ID != 20 {
		t.Errorf("overview todos = %+v, want todo ID 20", overview.Todos)
	}
	if len(overview.Carts) != 1 || overview.Carts[0].ID != 30 {
		t.Errorf("overview carts = %+v, want cart ID 30", overview.Carts)
	}
}

func TestGetPostDetailsToolFetchesCommentsAndAuthorConcurrently(t *testing.T) {
	var relatedRequestCount atomic.Int32
	relatedRequestsStarted := make(chan struct{})
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/posts/5" {
			fmt.Fprint(w, `{"id":5,"title":"A post","body":"Post body","userId":2}`)
			return
		}

		if relatedRequestCount.Add(1) == 2 {
			close(relatedRequestsStarted)
		}
		select {
		case <-relatedRequestsStarted:
		case <-time.After(3 * time.Second):
			http.Error(w, "related requests were not concurrent", http.StatusGatewayTimeout)
			return
		}

		switch r.URL.Path {
		case "/comments/post/5":
			fmt.Fprint(w, `{"comments":[{"id":15,"body":"A comment","postId":5,"user":{"id":3,"username":"reader"}}],"total":1,"skip":0,"limit":1}`)
		case "/users/2":
			fmt.Fprint(w, `{"id":2,"firstName":"Sam","lastName":"Lee","username":"author"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer apiServer.Close()

	apiClient := &api.Client{BaseURL: apiServer.URL, HTTPClient: apiServer.Client()}
	server := NewServer(apiClient)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{Name: "post-details-test", Version: "1.0.0"},
		nil,
	)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_post_details",
		Arguments: map[string]any{"postId": 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("get_post_details returned tool error: %v", result.GetError())
	}

	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var details PostDetails
	if err := json.Unmarshal(encoded, &details); err != nil {
		t.Fatal(err)
	}
	if details.Post == nil || details.Post.ID != 5 {
		t.Errorf("details post = %+v, want post ID 5", details.Post)
	}
	if details.Author == nil || details.Author.ID != 2 {
		t.Errorf("details author = %+v, want author ID 2", details.Author)
	}
	if len(details.Comments) != 1 || details.Comments[0].ID != 15 {
		t.Errorf("details comments = %+v, want comment ID 15", details.Comments)
	}
}

func TestCompareProductsToolPreservesOrderAndAvoidsDuplicateRequests(t *testing.T) {
	var requestCount atomic.Int32
	bothRequestsStarted := make(chan struct{})
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount.Add(1) == 2 {
			close(bothRequestsStarted)
		}
		select {
		case <-bothRequestsStarted:
		case <-time.After(3 * time.Second):
			http.Error(w, "product requests were not concurrent", http.StatusGatewayTimeout)
			return
		}

		switch r.URL.Path {
		case "/products/1":
			fmt.Fprint(w, `{"id":1,"title":"Unavailable","price":12.5,"rating":4.1,"category":"tools","stock":0}`)
		case "/products/2":
			fmt.Fprint(w, `{"id":2,"title":"Available","price":25,"rating":4.8,"category":"home","stock":8}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer apiServer.Close()

	apiClient := &api.Client{BaseURL: apiServer.URL, HTTPClient: apiServer.Client()}
	server := NewServer(apiClient)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{Name: "compare-products-test", Version: "1.0.0"},
		nil,
	)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	invalidResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "compare_products",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !invalidResult.IsError {
		t.Fatal("compare_products without productIds succeeded, want input validation error")
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("invalid input triggered %d upstream requests, want 0", got)
	}

	for _, arguments := range []any{
		map[string]any{"productIds": []int{2, 1, 2}},
		map[string]any{"productIds": []any{2, 1, 2}},
		json.RawMessage(`{"productIds":[2,1,2]}`),
	} {
		result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "compare_products",
			Arguments: arguments,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.IsError {
			t.Fatalf("compare_products with %T arguments returned tool error: %v", arguments, result.GetError())
		}
		encoded, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		var comparison CompareProductsResult
		if err := json.Unmarshal(encoded, &comparison); err != nil {
			t.Fatal(err)
		}
		if len(comparison.Products) != 3 || comparison.Products[0].ID != 2 || comparison.Products[1].ID != 1 || comparison.Products[2].ID != 2 {
			t.Errorf("comparison from %T arguments has unexpected products: %+v", arguments, comparison.Products)
		}
	}
	if got := requestCount.Load(); got != 6 {
		t.Errorf("upstream request count = %d, want 6 (two unique lookups for each argument encoding)", got)
	}
}

func TestCompareProductsRejectsEmptyOrInvalidIDs(t *testing.T) {
	for _, productIDs := range [][]int{nil, {}, {1, 0}} {
		if _, err := compareProducts(context.Background(), nil, productIDs); err == nil {
			t.Errorf("compareProducts(%v) error = nil, want validation error", productIDs)
		}
	}
}

func TestGetShoppingSummaryToolAggregatesCartsInOneRequest(t *testing.T) {
	var requestCount atomic.Int32
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if r.URL.Path != "/carts/user/9" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{
			"carts":[
				{"id":1,"userId":9,"total":120,"discountedTotal":108,"totalProducts":2,"totalQuantity":3,
				 "products":[
					{"id":10,"title":"Chair","price":50,"quantity":2,"total":100,"discountedTotal":90},
					{"id":11,"title":"Lamp","price":20,"quantity":1,"total":20,"discountedTotal":18}
				 ]},
				{"id":2,"userId":9,"total":30,"discountedTotal":27,"totalProducts":1,"totalQuantity":2,
				 "products":[
					{"id":12,"title":"Mug","price":15,"quantity":2,"total":30,"discountedTotal":27}
				 ]}
			],
			"total":2,"skip":0,"limit":2
		}`)
	}))
	defer apiServer.Close()

	apiClient := &api.Client{BaseURL: apiServer.URL, HTTPClient: apiServer.Client()}
	server := NewServer(apiClient)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(
		&mcp.Implementation{Name: "shopping-summary-test", Version: "1.0.0"},
		nil,
	)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_shopping_summary",
		Arguments: map[string]any{"userId": 9},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("get_shopping_summary returned tool error: %v", result.GetError())
	}
	if got := requestCount.Load(); got != 1 {
		t.Errorf("upstream request count = %d, want 1", got)
	}

	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var summary ShoppingSummary
	if err := json.Unmarshal(encoded, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.UserID != 9 || summary.CartCount != 2 || len(summary.Carts) != 2 {
		t.Errorf("summary identity/carts = %+v", summary)
	}
	if summary.TotalProducts != 3 || summary.TotalQuantity != 5 {
		t.Errorf("summary product counts = (%d, %d), want (3, 5)", summary.TotalProducts, summary.TotalQuantity)
	}
	if summary.TotalValue != 150 || summary.DiscountedValue != 135 {
		t.Errorf("summary totals = (%v, %v), want (150, 135)", summary.TotalValue, summary.DiscountedValue)
	}
	if len(summary.Carts[0].Products) != 2 || summary.Carts[0].Products[0].Quantity != 2 {
		t.Errorf("cart products/quantities were not returned: %+v", summary.Carts[0].Products)
	}
}

func TestGetShoppingSummaryReturnsEmptyCarts(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"carts":[],"total":0,"skip":0,"limit":0}`)
	}))
	defer apiServer.Close()

	apiClient := &api.Client{BaseURL: apiServer.URL, HTTPClient: apiServer.Client()}
	summary, err := getShoppingSummary(context.Background(), apiClient, 9)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Carts == nil || len(summary.Carts) != 0 {
		t.Errorf("empty carts = %#v, want non-nil empty slice", summary.Carts)
	}
	if summary.TotalValue != 0 || summary.DiscountedValue != 0 || summary.TotalQuantity != 0 {
		t.Errorf("empty cart totals = %+v, want zero totals", summary)
	}
}

func TestGetShoppingSummaryRejectsInvalidUserID(t *testing.T) {
	if _, err := getShoppingSummary(context.Background(), nil, 0); err == nil {
		t.Fatal("getShoppingSummary() error = nil, want validation error")
	}
}
