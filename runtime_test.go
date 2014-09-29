package notify

import "testing"

func TestRuntime(t *testing.T) {
	cases := [...]CallCases{{
		Watch: Call{P: "/github.com/rjeczalik/fakerpc", E: Delete},
		Expect: []Call{
			{T: TypeFanin},
			{T: TypeWatch, P: "/github.com/rjeczalik/fakerpc", E: Delete},
		},
	}}
	test.ExpectCalls(t, cases[:])
}
