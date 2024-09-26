package main

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	_ "gorm.io/gorm"
	"grpc-go-todo/entities"
	pb "grpc-go-todo/proto"
	"log"
	"net"
)

type server struct {
	pb.UnimplementedTodoServiceServer
	db *gorm.DB
}

type Config struct {
	Database struct {
		Host     string
		Port     string
		Username string
		Password string
		DbName   string
	}
}

func main() {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.SetConfigType("yaml")

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	// Unmarshal the config into our struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode into struct: %s", err)
	}

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.Database.Username,
		config.Database.Password,
		config.Database.Host,
		config.Database.Port,
		config.Database.DbName)
	fmt.Println(dsn)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	// Migrate the schema
	if err := db.AutoMigrate(&entities.Todo{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterTodoServiceServer(s, &server{db: db})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *server) CreateTodo(ctx context.Context, todo *pb.Todo) (*pb.Todo, error) {
	td := &pb.Todo{
		Title:       todo.Title,
		Description: todo.Description,
	}
	if err := s.db.Create(td).Error; err != nil {
		return nil, err
	} else {
		return &pb.Todo{Id: td.Id, Title: todo.Title, Description: todo.Description}, nil
	}

}

func (s *server) ReadTodo(ctx context.Context, id *pb.TodoId) (*pb.Todo, error) {
	todo := &pb.Todo{}

	if err := s.db.Find(&todo, id.Id).Error; err != nil {
		return nil, nil
	} else {
		return &pb.Todo{Id: todo.Id, Title: todo.Title, Description: todo.Description}, nil
	}
	return todo, nil
}

func (s *server) UpdateTodo(ctx context.Context, req *pb.Todo) (*pb.Todo, error) {
	var todo pb.Todo
	if err := s.db.First(&todo, req.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("Todo with ID %d not found", req.Id)
		}
		return nil, nil
	}
	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["Title"] = req.Title
	}
	if req.Description != "" {
		updates["Description"] = req.Description
	}
	if err := s.db.Model(&todo).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &pb.Todo{Id: todo.Id, Title: todo.Title, Description: todo.Description}, nil
}

func (s *server) DeleteTodo(ctx context.Context, id *pb.TodoId) (*pb.TodoId, error) {
	var todo pb.Todo

	if err := s.db.First(&todo, id.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("Todo with ID %d not found", id.Id)
		}
		return nil, err
	}
	if err := s.db.Delete(&todo, id.Id).Error; err != nil {
		return nil, err
	}
	return id, nil
}
