package network

import "github.com/leighmacdonald/gbans/internal/network/v1/networkv1connect"

type BlocklistService struct {
	networkv1connect.UnimplementedNetworkServiceHandler

	networks Networks
}

func NewBlocklistService(networks Networks) BlocklistService {
	return BlocklistService{networks: networks}
}
