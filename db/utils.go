package db

var (
	NamespaceAggregatorTrie                               = []byte("at")
	NamespaceValidatorTrie                                = []byte("vt")
	NamespaceRollupBlockTrie                              = []byte("rbt")
	NamespaceTokenAddressToTokenIndex                     = []byte("tatti")
	NamespaceTokenIndexToTokenAddress                     = []byte("titta")
	NamespaceMainchainTokenAddressToSidechainTokenAddress = []byte("mtatst")
	NamespaceAccountAddressToKey                          = []byte("aatk")
	NamespaceLastKey                                      = []byte("lk")
	NamespaceKeyToAccountInfo                             = []byte("ktai")
	NamespaceRollupBlockNumber                            = []byte("rbn")
	EmptyKey                                              = []byte{}
	Separator                                             = []byte("|")
)

func PrependNamespace(namespace []byte, key []byte) []byte {
	if namespace != nil {
		return append(append(namespace, Separator...), key...)
	}
	return key
}

func ConvNilToBytes(byteArray []byte) []byte {
	if byteArray == nil {
		return []byte{}
	}
	return byteArray
}
