package clustermanager

func waitOrError(tc chan bool, ec chan error, numProcPtr *int) error {
	numProcs := *numProcPtr
	for numProcs > 0 {
		select {
		case err := <-ec:
			return err
		case <-tc:
			numProcs--
		}
	}

	return nil
}
