# MCP Inspector Input Guide

This document lists the JSON arguments accepted by every MCP tool currently
registered by this project. In MCP Inspector, open **Tools**, select a tool,
enter the example JSON object in its arguments/input box, and run it. Do not
include the tool name in the JSON; Inspector already knows which tool you chose.

All examples below are argument objects. You can omit optional fields.

## Collection options

These fields are accepted by `list_products`, `list_users`, `list_posts`,
`list_comments`, `list_carts`, `list_todos`, `list_recipes`,
`get_products_by_category`, `get_posts_by_user`, `get_todos_by_user`,
`get_comments_by_post`, `get_carts_by_user`, `search_products`, `search_users`,
`search_posts`, and `search_recipes`:

| Field | Type | Meaning |
|---|---|---|
| `limit` | integer | Maximum number of records to return. Omit or use `0` for the endpoint default. |
| `skip` | integer | Number of records to skip. Must not be negative. |
| `sortBy` | string | Field name to sort by. |
| `order` | string | `asc` or `desc`; requires `sortBy`. |
| `select` | array of strings | Record fields to include, for example `["title","price"]`. |

`limit` must not be negative. Sorting, ordering, and field selection are
forwarded to DummyJSON, so available fields depend on the selected collection.
If no options are supplied, users, posts, comments, and carts keep the server's
existing default limit of 10; products use the DummyJSON endpoint default.

Example:

```json
{
  "limit": 5,
  "skip": 10,
  "sortBy": "price",
  "order": "desc",
  "select": ["title", "price", "rating"]
}
```

## Tool inputs

### `list_products`

Lists products. All collection options are optional.

```json
{
  "limit": 5,
  "skip": 0,
  "sortBy": "price",
  "order": "asc",
  "select": ["title", "price", "rating", "category", "stock"]
}
```

To use the endpoint defaults, submit `{}`.

### `get_product`

Gets one product. `id` must be a positive integer.

```json
{"id": 1}
```

### `get_products_by_category`

Lists products in a category. `category` must be a non-empty category slug;
collection options are optional.

```json
{
  "category": "smartphones",
  "limit": 5,
  "skip": 0,
  "sortBy": "price",
  "order": "asc",
  "select": ["title", "price", "stock"]
}
```

### `compare_products`

Compares the products whose IDs are in `productIds`. Supply a non-empty array of
positive integers. The result preserves this order; repeated IDs are allowed.

```json
{"productIds": [1, 2, 3]}
```

### `search_products`

Searches products. `searchTerm` must contain non-whitespace text.

```json
{"searchTerm": "phone"}
```

### `list_users`

Lists users. Accepts the collection options described above.

```json
{
  "limit": 5,
  "skip": 0,
  "sortBy": "id",
  "order": "asc",
  "select": ["id", "firstName", "lastName", "username"]
}
```

### `get_user`

Gets one user. `id` must be a positive integer.

```json
{"id": 1}
```

### `search_users`

Searches users by a keyword. `searchTerm` must contain non-whitespace text.
Optional collection options can limit or sort the matches.

```json
{
  "searchTerm": "Emily",
  "limit": 5,
  "skip": 0,
  "select": ["id", "firstName", "lastName", "username"]
}
```

### `get_user_overview`

Gets a user's profile, posts, todos, and carts. `userId` must be a positive
integer.

```json
{"userId": 1}
```

### `list_posts`

Lists posts. Accepts the collection options described above.

```json
{
  "limit": 5,
  "skip": 0,
  "sortBy": "id",
  "order": "asc",
  "select": ["id", "title", "body", "userId"]
}
```

### `get_post`

Gets one post. `id` must be a positive integer.

```json
{"id": 1}
```

### `get_post_details`

Gets a post, its comments, and author information. `postId` must be a positive
integer.

```json
{"postId": 1}
```

### `search_posts`

Searches posts. `searchTerm` must contain non-whitespace text.

```json
{"searchTerm": "love"}
```

### `get_posts_by_user`

Gets posts written by a user. `userId` must be a positive integer; collection
options are optional.

```json
{
  "userId": 1,
  "limit": 5,
  "skip": 0,
  "sortBy": "id",
  "order": "asc",
  "select": ["id", "title", "userId"]
}
```

### `list_comments`

Lists comments. Accepts the collection options described above.

```json
{
  "limit": 5,
  "skip": 0,
  "sortBy": "id",
  "order": "asc",
  "select": ["id", "body", "postId", "likes"]
}
```

### `get_comments_by_post`

Gets comments for one post. `postId` must be a positive integer. Collection
options are also accepted.

```json
{
  "postId": 1,
  "limit": 5,
  "skip": 0,
  "sortBy": "id",
  "order": "asc",
  "select": ["id", "body", "postId", "likes"]
}
```

### `get_comments_by_user`

Gets comments written by a user. `userId` must be a positive integer. DummyJSON
does not provide a comments-by-user endpoint, so this tool fetches the comments
collection and filters by the embedded comment author. It supports `limit`,
`skip`, and sorting by `id`, `postId`, or `likes`; field selection is not
supported for this local filter.

```json
{
  "userId": 1,
  "limit": 10,
  "skip": 0,
  "sortBy": "likes",
  "order": "desc"
}
```

### `list_todos`

Lists todos. Accepts the collection options described above.

```json
{
  "limit": 5,
  "skip": 0,
  "sortBy": "id",
  "order": "asc",
  "select": ["id", "todo", "completed", "userId"]
}
```

### `get_todo`

Gets one todo. `id` must be a positive integer.

```json
{"id": 1}
```

### `get_todos_by_user`

Gets todos assigned to a user. `userId` must be a positive integer; collection
options are optional.

```json
{
  "userId": 1,
  "limit": 5,
  "skip": 0,
  "select": ["id", "todo", "completed"]
}
```

### `list_recipes`

Lists recipes. Accepts the collection options described above.

```json
{
  "limit": 5,
  "skip": 0,
  "sortBy": "rating",
  "order": "desc",
  "select": ["id", "name", "cuisine", "rating", "mealType"]
}
```

### `get_recipe`

Gets one recipe. `id` must be a positive integer.

```json
{"id": 1}
```

### `search_recipes`

Searches recipes by a keyword. `searchTerm` must contain non-whitespace text;
collection options are optional.

```json
{
  "searchTerm": "pizza",
  "limit": 5,
  "skip": 0,
  "select": ["id", "name", "ingredients", "instructions", "rating"]
}
```

### `list_carts`

Lists carts. Accepts the collection options described above.

```json
{
  "limit": 5,
  "skip": 0,
  "sortBy": "id",
  "order": "asc",
  "select": ["id", "userId", "total", "discountedTotal"]
}
```

### `get_cart`

Gets one cart. `id` must be a positive integer.

```json
{"id": 1}
```

### `get_carts_by_user`

Gets carts for one user. `userId` must be a positive integer. Collection
options are also accepted.

```json
{
  "userId": 1,
  "limit": 5,
  "skip": 0,
  "sortBy": "id",
  "order": "asc",
  "select": ["id", "userId", "total", "discountedTotal"]
}
```

### `get_shopping_summary`

Summarizes a user's carts, products, quantities, and totals. `userId` must be a
positive integer.

```json
{"userId": 1}
```

## Common input mistakes

- Pass a JSON **object** in the Inspector input box, such as
  `{"id":1}`—not a bare number or a quoted JSON string.
- Use the exact property spelling and casing shown above: `productIds`,
  `searchTerm`, `postId`, and `userId`.
- IDs are positive integers. A validly formatted ID may still produce an API
  error if DummyJSON has no matching record.
- `order` must be `asc` or `desc`, and requires `sortBy`.
- `select` is an array of strings, not a comma-separated string. For example:
  `"select": ["title", "price"]`.
- Search terms must not be empty or whitespace-only.

## Keeping this guide current

Whenever a tool is added, removed, renamed, or its input fields change, update
this guide in the same change. Check the `mcp.AddTool` registrations and their
input structs in `mcpserver/server.go`, then update the tool list and example
here. Run `go test ./...` to confirm that the documented tool names still match
the server's registered tools.