package room

type Room struct {
	ID       int64
	Name     string
	Capacity int
	IsActive bool
}

type CreateRoomInput struct {
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}
