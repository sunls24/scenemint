package quota

import "context"

func (s *Store) Status(_ context.Context, req Request) (StatusResponse, error) {
	resp, err := s.Get(req.Fingerprint)
	if err != nil {
		return StatusResponse{}, ResponseError(err)
	}
	return resp, nil
}

func (s *Store) CheckIn(_ context.Context, req Request) (StatusResponse, error) {
	resp, err := s.ApplyCheckIn(req.Fingerprint)
	if err != nil {
		return StatusResponse{}, ResponseError(err)
	}
	return resp, nil
}
