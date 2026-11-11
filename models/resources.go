package models

// USERS

type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Age       int    `json:"age"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Phone     string `json:"phone"`
}

type UsersResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
	Skip  int    `json:"skip"`
	Limit int    `json:"limit"`
}

// POSTS

type Post struct {
	ID     int      `json:"id"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Tags   []string `json:"tags"`
	UserID int      `json:"userId"`
}

type PostsResponse struct {
	Posts []Post `json:"posts"`
	Total int    `json:"total"`
	Skip  int    `json:"skip"`
	Limit int    `json:"limit"`
}

// COMMENTS

type CommentUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type Comment struct {
	ID     int         `json:"id"`
	Body   string      `json:"body"`
	PostID int         `json:"postId"`
	Likes  int         `json:"likes"`
	User   CommentUser `json:"user"`
}

type CommentsResponse struct {
	Comments []Comment `json:"comments"`
	Total    int       `json:"total"`
	Skip     int       `json:"skip"`
	Limit    int       `json:"limit"`
}

// A product inside a cart.
type CartProduct struct {
	ID                 int     `json:"id"`
	Title              string  `json:"title"`
	Price              float64 `json:"price"`
	Quantity           int     `json:"quantity"`
	Total              float64 `json:"total"`
	DiscountPercentage float64 `json:"discountPercentage"`
	DiscountedTotal    float64 `json:"discountedTotal"`
	Thumbnail          string  `json:"thumbnail"`
}

// Represents one shopping cart.
type Cart struct {
	ID              int           `json:"id"`
	Products        []CartProduct `json:"products"`
	Total           float64       `json:"total"`
	DiscountedTotal float64       `json:"discountedTotal"`
	UserID          int           `json:"userId"`
	TotalProducts   int           `json:"totalProducts"`
	TotalQuantity   int           `json:"totalQuantity"`
}

// Represents the response when fetching multiple carts.
type CartsResponse struct {
	Carts []Cart `json:"carts"`
	Total int    `json:"total"`
	Skip  int    `json:"skip"`
	Limit int    `json:"limit"`
}

// TODO

type Todo struct {
	ID        int    `json:"id"`
	Todo      string `json:"todo"`
	Completed bool   `json:"completed"`
	UserID    int    `json:"userId"`
}

type TodosResponse struct {
	Todos []Todo `json:"todos"`
	Total int    `json:"total"`
	Skip  int    `json:"skip"`
	Limit int    `json:"limit"`
}
