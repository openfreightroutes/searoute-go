package searoute

func segmentIntersectsBBox(a, b [2]float64, bb [4]float64) bool {
	dx := b[0] - a[0]
	dy := b[1] - a[1]
	t0, t1 := 0.0, 1.0
	p := [4]float64{-dx, dx, -dy, dy}
	q := [4]float64{a[0] - bb[0], bb[2] - a[0], a[1] - bb[1], bb[3] - a[1]}
	for i := range p {
		if p[i] == 0 {
			if q[i] < 0 {
				return false
			}
			continue
		}
		r := q[i] / p[i]
		if p[i] < 0 {
			if r > t1 {
				return false
			}
			if r > t0 {
				t0 = r
			}
		} else {
			if r < t0 {
				return false
			}
			if r < t1 {
				t1 = r
			}
		}
	}
	return true
}
