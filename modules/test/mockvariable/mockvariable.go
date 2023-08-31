// SPDX-License-Identifier: MIT

package mockvariable

func Value[T any](p *T, v T) (reset func()) {
	old := *p
	*p = v
	return func() { *p = old }
}
