//sheared maeesage format between the broker and the worker.
package stub

type ComputeRequest struct{
	StartY int
	EndY int
	CurrentMap [][]uint8
	ImageHeight int
	ImageWidth int
}

type ComputeResponse struct{
	Calc  [][]uint8
}

