package kb

// rewriteAnnotationContentPathsAfterMove updates anchor content_path values after a node move.
func (s *Store) rewriteAnnotationContentPathsAfterMove(basePath, oldNodePath, newNodePath string) error {
	file, err := s.readAnnotationsFile(basePath, newNodePath)
	if err != nil {
		return err
	}
	if len(file.Notes) == 0 {
		return nil
	}
	changed := false
	for i := range file.Notes {
		if file.Notes[i].Anchor == nil {
			continue
		}
		updated, ok := remapAnnotationContentPath(file.Notes[i].Anchor.ContentPath, oldNodePath, newNodePath)
		if ok && updated != file.Notes[i].Anchor.ContentPath {
			file.Notes[i].Anchor.ContentPath = updated
			changed = true
		}
	}
	if !changed {
		return nil
	}

	return s.writeAnnotationsFile(basePath, newNodePath, file)
}
