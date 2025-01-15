package user

import "time"

type CreateUserRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Name       string `json:"name" validate:"required,lte=50"`
	Password   string `json:"password" validate:"required,gte=12"`
	PublicKey  string `json:"publicKey" validate:"required"`
	PrivateKey string `json:"privateKey" validate:"required"`
	Iv         string `json:"iv" validate:"required,lte=16"`
}

type CreateUserResponse struct {
	Id        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updateAt"`
	Email     string `json:"email"`
}

type GetUserResponse struct {
	Id         string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Bio        *string   `json:"bio"`
	Iv         string    `json:"iv"`
	ProfilePic *string   `json:"profilePic"`
	PublicKey  string    `json:"publicKey"`
	PrivateKey string    `json:"privateKey"`
}

type CheckIfuserExistsRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type UpdateUserRequest struct {
	Name           *string `json:"name" validate:"omitempty,lte=50"`
	Bio            *string `json:"bio" validate:"omitempty,lte=1000"`
	ProfilePicture *string `json:"profilePicture" validate:"omitempty,lte=2048"`
}

type UpdateUserResponse struct {
	Id             string    `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updateAt"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	Bio            *string   `json:"bio"`
	ProfilePicture *string   `json:"profilePicture"`
}
