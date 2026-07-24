package models

// SelectorType define cómo se eligen sabores al pedir:
//
//   - none: la categoría no admite sabores.
//   - single_select: cada sabor se elige a lo sumo una vez (cantidad 1).
//     min_selection / max_selection = cuántos sabores distintos debe/puede
//     elegir el cliente (ej. pizza: 5 sabores, min 2, max 4).
//   - multi_select: se puede pedir cantidad N del mismo sabor
//     (ej. refrescos: 3 de uva y 2 de limón).
const (
	SelectorNone         = "none"
	SelectorSingleSelect = "single_select"
	SelectorMultiSelect  = "multi_select"
)

func IsValidSelectorType(value string) bool {
	return value == SelectorNone || value == SelectorSingleSelect || value == SelectorMultiSelect
}

func SupportsFlavorSelection(value string) bool {
	return value == SelectorSingleSelect || value == SelectorMultiSelect
}

func NormalizeSelectorType(value string) string {
	if value == "" {
		return SelectorNone
	}
	return value
}
