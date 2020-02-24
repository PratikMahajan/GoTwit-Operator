package controller

import (
	"github.com/pratikmahajan/GoTwit-Operator/pkg/controller/gotwit"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, gotwit.Add)
}
