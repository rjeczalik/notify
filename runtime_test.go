package notify

import "testing"

func TestRuntime(t *testing.T) {
	cases := [...]CallCases{
		0: {
			Watch: Call{P: "/github.com/rjeczalik/fakerpc", E: Delete},
			Expect: []Call{
				{T: TypeFanin},
				{T: TypeWatch, P: "/github.com/rjeczalik/fakerpc", E: Delete},
			},
		},
		1: {
			Watch: Call{P: "/github.com/rjeczalik/fakerpc/...", E: Delete},
			Expect: []Call{
				{T: TypeFanin},
				{T: TypeWatch, P: "/github.com/rjeczalik/fakerpc", E: Delete},
				{T: TypeWatch, P: "/github.com/rjeczalik/fakerpc/cli", E: Delete},
				{T: TypeWatch, P: "/github.com/rjeczalik/fakerpc/cmd", E: Delete},
			},
		},
		2: {
			Watch: Call{P: "/github.com/rjeczalik/fakerpc/LICENSE", E: Write},
			Expect: []Call{
				{T: TypeFanin},
				{T: TypeWatch, P: "/github.com/rjeczalik/fakerpc/LICENSE", E: Write},
			},
		}}
	// TODO(rjeczalik): Enable rest of test-cases.
	test.ExpectCalls(t, cases[:1])
}
