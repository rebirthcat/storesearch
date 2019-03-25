package distscore


type DistScoreCriteria interface {
	Score(lat float64,lng float64) float32
}