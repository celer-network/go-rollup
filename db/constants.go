package db

var (
	NamespaceTrie                                         = []byte("trie")
	NamespaceTokenAddressToTokenIndex                     = []byte("tatti")
	NamespaceMainchainTokenAddressToSidechainTokenAddress = []byte("mtatst")
	NamespaceAccountAddressToKey                          = []byte("aatk")
	NamespaceLastKey                                      = []byte("lk")
	NamespaceKeyToAccountInfo                             = []byte("ktai")
	EmptyKey                                              = []byte{}
	separator                                             = []byte("|")
)
