package orderbuilder

// GetSignatureType returns the signature type of the order builder
// Helper method to expose signature type for client use
func (ob *OrderBuilder) GetSignatureType() int {
	return int(ob.sigType)
}