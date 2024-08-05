package main

import (
	"errors"
	"fmt"
	"log"
	"sync/atomic"
)

type Service interface {
	Transaction() error
	Compensate() error
	GetName() string
}

type Saga struct {
	Service []Service

	isExecutable        atomic.Bool
	successfulTransfers int
	errors              map[string]error
}

func NewSaga(service ...Service) *Saga {
	return &Saga{
		Service: service,
	}
}

func (s *Saga) ExecuteTransaction() error {
	if isSwap := s.isExecutable.CompareAndSwap(false, true); !isSwap {
		return errors.New("transaction is already in execute")
	}

	for _, service := range s.Service {
		if err := service.Transaction(); err != nil {
			s.compensate()
			return err
		}

		s.successfulTransfers++
	}

	return nil
}

func (s *Saga) compensate() {
	for i := s.successfulTransfers; i > 0; i-- {
		if err := s.Service[i].Compensate(); err != nil {
			s.errors[s.Service[i].GetName()] = err
		}
	}
}

func (s *Saga) GetErrors() map[string]error {
	return s.errors
}

type EmptyService struct {
	Name   string
	Result string
}

func NewEmptyService(name string) *EmptyService {
	return &EmptyService{
		Name: name,
	}
}

func (e *EmptyService) Transaction() error {
	fmt.Printf("%s: Выполнение транзакции\n", e.Name)
	e.Result = fmt.Sprintf("resp: %s", e.Name)
	return nil
}

func (e *EmptyService) Compensate() error {
	fmt.Printf("%s: Компенсация транзакции\n", e.Name)
	return nil
}

func (e *EmptyService) GetName() string {
	return e.Name
}

type FailedService struct {
	Name string
}

func NewFailedService(name string) *FailedService {
	return &FailedService{
		Name: name,
	}
}

func (e *FailedService) Transaction() error {
	fmt.Printf("%s: Выполнение транзакции\n", e.Name)
	return errors.New("something went wrong")
}

func (e *FailedService) Compensate() error {
	fmt.Printf("%s: Компенсация транзакции\n", e.Name)
	return nil
}

func (e *FailedService) GetName() string {
	return e.Name
}

func main() {
	s1 := NewEmptyService("Service-1")
	s2 := NewEmptyService("Service-2")
	s3 := NewEmptyService("Service-3")

	s4 := NewFailedService("Service-4-Failed")
	saga := NewSaga(s1, s2, s3, s4)

	//saga := NewSaga(s1, s2, s3)

	if err := saga.ExecuteTransaction(); err != nil {
		log.Fatal(err)
	}

	if errs := saga.errors; len(errs) > 0 {
		for _, err := range errs {
			fmt.Println(err)
		}
		return
	}

	fmt.Println(s1.Result)
	fmt.Println(s2.Result)
	fmt.Println(s3.Result)
}
