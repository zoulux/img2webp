package analyze

func Classify(f Features) Kind {
	if f.HasAlpha {
		return KindTransparentGraphic
	}
	if f.UniqueColorDensity < 0.20 && f.TextureScore < 0.25 {
		return KindGraphic
	}
	return KindPhoto
}
