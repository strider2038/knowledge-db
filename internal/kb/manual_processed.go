package kb

import "maps"

// ManualProcessedEffective returns whether the node is marked as manually processed.
// Missing or null key is treated as false. Non-boolean values are treated as false for API output;
// validation rejects such values in stored files.
func ManualProcessedEffective(meta map[string]any) bool {
	if meta == nil {
		return false
	}
	v, ok := meta["manual_processed"]
	if !ok || v == nil {
		return false
	}
	b, ok := v.(bool)

	return ok && b
}

// NormalizeNodeMetadataForAPI returns a shallow copy of meta with normalized manual_processed (bool).
func NormalizeNodeMetadataForAPI(meta map[string]any) map[string]any {
	if meta == nil {
		return map[string]any{"manual_processed": false}
	}
	out := maps.Clone(meta)
	out["manual_processed"] = ManualProcessedEffective(meta)

	return out
}
