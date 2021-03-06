package matrix

import (
	"fmt"
	"github.com/gonum/matrix/mat64"
	"math"
	"math/rand"
)

// Distance calculations we support
type DistType int

const (
	EuclideanDist DistType = iota
)

// A two dimensional array with some special functionality
type Matrix interface {
	NDArray

	// Set the values of the items on a given column
	ColSet(col int, values []float64)

	// Get a particular column for read-only access. May or may not be a copy.
	Col(col int) []float64

	// Get the number of columns
	Cols() int

	// Get a column vector containing the main diagonal elements of the matrix
	Diag() Matrix

	// Treat the rows as points, and get the pairwise distance between them.
	// Returns a distance matrix D such that D_i,j is the distance between
	// rows i and j.
	Dist(t DistType) Matrix

	// Get the matrix inverse
	Inverse() (Matrix, error)

	// Solve for x, where ax = b and a is `this`.
	LDivide(b Matrix) Matrix

	// Get the result of matrix multiplication between this and some other
	// matrices. Matrix dimensions must be aligned correctly for multiplication.
	// If A is m x p and B is p x n, then C = A.MProd(B) is the m x n matrix
	// with C[i, j] = \sum_{k=1}^p A[i,k] * B[k,j].
	MProd(others ...Matrix) Matrix

	// Get the matrix norm of the specified ordinality (1, 2, infinity, ...)
	Norm(ord float64) float64

	// Set the values of the items on a given row
	RowSet(row int, values []float64)

	// Get a particular column for read-only access. May or may not be a copy.
	Row(row int) []float64

	// Get the number of rows
	Rows() int

	// Return the same matrix, but with axes transposed. The same data is used,
	// for speed and memory efficiency. Use Copy() to create a new array.
	T() Matrix

	// Return a sparse coo copy of the matrix. The method will panic
	// if any off-diagonal elements are nonzero.
	SparseCoo() Matrix

	// Return a sparse diag copy of the matrix. The method will panic
	// if any off-diagonal elements are nonzero.
	SparseDiag() Matrix
}

// Create a square matrix with the specified elements on the main diagonal, and
// zero elsewhere.
func Diag(diag ...float64) Matrix {
	array := SparseDiag(len(diag), len(diag))
	for i, v := range diag {
		array.ItemSet(v, i, i)
	}
	return array
}

// Create a square sparse identity matrix of the specified dimensionality.
func Eye(size int) Matrix {
	diag := make([]float64, size)
	for i := 0; i < size; i++ {
		diag[i] = 1.0
	}
	return Diag(diag...)
}

// Create a matrix from literal data
func M(rows, cols int, array ...float64) Matrix {
	return A([]int{rows, cols}, array...).M()
}

// Create a matrix from literal data and the provided shape
func M2(array ...[]float64) Matrix {
	return A2(array...).M()
}

// Create a sparse matrix of the specified dimensionality. This matrix will be
// stored in coordinate format: each entry is stored as a (x, y, value) triple.
// The first len(array) elements of the matrix will be initialized to the
// corresponding nonzero values of array.
func SparseCoo(rows, cols int, array ...float64) Matrix {
	m := &sparseCooF64Matrix{
		shape:  []int{rows, cols},
		values: make([]map[int]float64, rows),
	}
	for i := 0; i < rows; i++ {
		m.values[i] = make(map[int]float64)
	}
	for idx, val := range array {
		if val != 0 {
			m.ItemSet(val, flatToNd(m.shape, idx)...)
		}
	}
	return m
}

// Create a sparse matrix of the specified dimensionality. This matrix will be
// stored in diagonal format: the main diagonal is stored as a []float64, and
// all off-diagonal values are zero. The matrix is initialized from diag, or
// to all zeros.
func SparseDiag(rows, cols int, diag ...float64) Matrix {
	if len(diag) > rows || len(diag) > cols {
		panic(fmt.Sprintf("Can't use %d diag elements in a %dx%d matrix", len(diag), rows, cols))
	}
	size := rows
	if cols < rows {
		size = cols
	}
	array := &sparseDiagF64Matrix{
		shape: []int{rows, cols},
		diag:  make([]float64, size),
	}
	for pos, v := range diag {
		array.diag[pos] = v
	}
	return array
}

// Create a sparse coo matrix, randomly populated so that approximately
// density * rows * cols cells are filled with random values uniformly
// distributed in [0,1). Note that if density is close to 1, this function may
// be extremely slow.
func SparseRand(rows, cols int, density float64) Matrix {
	if density < 0 || density >= 1 {
		panic(fmt.Sprintf("Can't create a SparseRand matrix: density %f should be in [0, 1)", density))
	}
	matrix := SparseCoo(rows, cols)
	shape := []int{rows, cols}
	size := rows * cols
	count := int(float64(size) * density)
	for i := 0; i < count; i++ {
		for {
			coord := flatToNd(shape, rand.Intn(size))
			if matrix.Item(coord...) == 0 {
				matrix.ItemSet(rand.Float64(), coord...)
				break
			}
		}
	}
	return matrix
}

// Create a sparse coo matrix, randomly populated so that approximately
// density * rows * cols cells are filled with random values in the range
// [-math.MaxFloat64, +math.MaxFloat64] distributed on the standard Normal
// distribution.  Note that if density is close to 1, this function may
// be extremely slow.
func SparseRandN(rows, cols int, density float64) Matrix {
	if density < 0 || density >= 1 {
		panic(fmt.Sprintf("Can't create a SparseRandN matrix: density %f should be in [0, 1)", density))
	}
	matrix := SparseCoo(rows, cols)
	shape := []int{rows, cols}
	size := rows * cols
	count := int(float64(size) * density)
	for i := 0; i < count; i++ {
		for {
			coord := flatToNd(shape, rand.Intn(size))
			if matrix.Item(coord...) == 0 {
				matrix.ItemSet(rand.NormFloat64(), coord...)
				break
			}
		}
	}
	return matrix
}

// Convert our matrix type to mat64's matrix type
func ToMat64(m Matrix) *mat64.Dense {
	return mat64.NewDense(m.Rows(), m.Cols(), m.Array())
}

// Convert mat64's matrix type to our matrix type
func ToMatrix(m mat64.Matrix) Matrix {
	rows, cols := m.Dims()
	array := &denseF64Array{
		shape: []int{rows, cols},
		array: make([]float64, rows*cols),
	}
	for i0 := 0; i0 < rows; i0++ {
		for i1 := 0; i1 < cols; i1++ {
			array.ItemSet(m.At(i0, i1), i0, i1)
		}
	}
	return array
}

// Get the matrix inverse
func Inverse(a Matrix) (Matrix, error) {
	inv, err := mat64.Inverse(ToMat64(a))
	if err != nil {
		return nil, err
	}
	return ToMatrix(inv), nil
}

// Solve for x, where ax = b.
func LDivide(a, b Matrix) Matrix {
	var x mat64.Dense
	err := x.Solve(ToMat64(a), ToMat64(b))
	if err != nil {
		return WithValue(math.NaN(), a.Shape()[0], b.Shape()[1]).M()
	}
	return ToMatrix(&x)
}

// Get the matrix norm of the specified ordinality (1, 2, infinity, ...)
func Norm(m Matrix, ord float64) float64 {
	return ToMat64(m).Norm(ord)
}

// Solve is an alias for LDivide
func Solve(a, b Matrix) Matrix {
	return LDivide(a, b)
}
