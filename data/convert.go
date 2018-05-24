// GOOOOOOOO

package data

func PointerIDsToStrings(ids []PointerID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, id.String())
	}
	return result
}

func SnapshotIDs(ids ...string) []SnapshotID {
	return StringsToSnapshotIDs(ids)
}

func SnapshotIDsToStrings(ids []SnapshotID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, id.String())
	}
	return result
}

func StringsToPointerIDs(ids []string) ([]PointerID, error) {
	result := make([]PointerID, 0, len(ids))
	for _, id := range ids {
		ptr, err := ParsePointerID(id)
		if err != nil {
			return nil, err
		}
		result = append(result, ptr)
	}
	return result, nil
}

func StringsToSnapshotIDs(ids []string) []SnapshotID {
	result := make([]SnapshotID, 0, len(ids))
	for _, id := range ids {
		result = append(result, ParseSnapshotID(id))
	}
	return result
}

func PointerRevsToInts(revs []PointerRev) []int64 {
	result := make([]int64, 0, len(revs))
	for _, rev := range revs {
		result = append(result, int64(rev))
	}
	return result
}

func IntsToPointerRevs(revs []int64) []PointerRev {
	result := make([]PointerRev, 0, len(revs))
	for _, rev := range revs {
		result = append(result, PointerRev(rev))
	}
	return result
}

func PointerIDsToSet(ids []PointerID) map[PointerID]bool {
	result := make(map[PointerID]bool, len(ids))
	for _, id := range ids {
		result[id] = true
	}
	return result
}

func SetToPointerIDs(ids map[PointerID]bool) []PointerID {
	result := make([]PointerID, 0, len(ids))
	for id, _ := range ids {
		result = append(result, id)
	}
	return result
}

func SnapshotIDsEqual(ids, ids2 []SnapshotID) bool {
	if len(ids) != len(ids2) {
		return false
	}
	for i, id := range ids {
		id2 := ids[i]
		if id != id2 {
			return false
		}
	}
	return true
}
