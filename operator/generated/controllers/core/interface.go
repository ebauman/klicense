/*
Copyright 2022.

All Rights Reserved
*/
// Code generated by main. DO NOT EDIT.

package core

import (
	v1 "github.com/ebauman/klicense/operator/generated/controllers/core/v1"
	"github.com/rancher/lasso/pkg/controller"
)

type Interface interface {
	V1() v1.Interface
}

type group struct {
	controllerFactory controller.SharedControllerFactory
}

// New returns a new Interface.
func New(controllerFactory controller.SharedControllerFactory) Interface {
	return &group{
		controllerFactory: controllerFactory,
	}
}

func (g *group) V1() v1.Interface {
	return v1.New(g.controllerFactory)
}
