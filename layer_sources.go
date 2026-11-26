package main

const layerL1 = `#usda 1.0
(
    defaultPrim = "Stage"
    layerId = "L1-base"
    layerStrength = 10
)

# Weak base layer: it defines the namespace and supplies fallback colors.
def Xform "Stage" {
    def Mesh "PanelA" {
        color3f inputs:displayColor = (0.6, 0.62, 0.68)
        def Sphere "Knob" {
            color3f inputs:displayColor = (0.14, 0.15, 0.19)
        }
    }

    def Mesh "PanelB" {
        color3f inputs:displayColor = (0.6, 0.62, 0.68)
    }

    def Mesh "PanelC" {
        color3f inputs:displayColor = (0.6, 0.62, 0.68)
        def Sphere "Badge" {
            color3f inputs:displayColor = (0.58, 0.6, 0.66)
        }
    }

    def Mesh "PanelD" {
        color3f inputs:displayColor = (0.47, 0.56, 0.52)
        def Sphere "Tag" {
            color3f inputs:displayColor = (0.48, 0.5, 0.55)
        }
    }

    def Mesh "StackProbe" {
        color3f inputs:displayColor = (0.42, 0.43, 0.5)
    }
}
`

const layerL2 = `#usda 1.0
(
    defaultPrim = "Stage"
    layerId = "L2-warm"
    layerStrength = 25
)

# Warm layer. Every prim is an over: no new namespace is introduced.
over Xform "Stage" {
    over Mesh "PanelA" {
        color3f inputs:displayColor = (0.9, 0.27, 0.22)
        over Sphere "Knob" {
            color3f inputs:displayColor = (1, 0.72, 0.25)
        }
    }

    over Mesh "PanelB" {
        color3f inputs:displayColor = (0.95, 0.55, 0.16)
    }

    # This local over is intentionally nested below another over. The child
    # still resolves to /Stage/PanelD/Tag, never /Stage/PanelD/Tag/Tag.
    over Mesh "PanelD" {
        over Sphere "Tag" {
            color3f inputs:displayColor = (0.98, 0.8, 0.3)
        }
    }

    over Mesh "StackProbe" {
        color3f inputs:displayColor = (0.95, 0.42, 0.46)
    }
}
`

const layerL3 = `#usda 1.0
(
    defaultPrim = "Stage"
    layerId = "L3-cool"
    layerStrength = 30
)

# Cool layer. The nested over under PanelA proves that path composition is
# relative: it targets /Stage/PanelA/Knob and not a relocated child node.
over Xform "Stage" {
    over Mesh "PanelA" {
        over Sphere "Knob" {
            color3f inputs:displayColor = (0.3, 0.92, 0.88)
        }
    }

    over Mesh "PanelB" {
        color3f inputs:displayColor = (0.1, 0.76, 0.66)
    }

    over Mesh "PanelC" {
        color3f inputs:displayColor = (0.2, 0.45, 0.92)
    }

    over Mesh "StackProbe" {
        color3f inputs:displayColor = (0.36, 0.34, 0.92)
    }
}
`

const layerL4 = `#usda 1.0
(
    defaultPrim = "Stage"
    layerId = "L4-spot"
    layerStrength = 40
)

# Strongest layer. Local over statements only add opinions to existing paths.
over Xform "Stage" {
    over Mesh "PanelC" {
        color3f inputs:displayColor = (0.98, 0.8, 0.18)
        over Sphere "Badge" {
            color3f inputs:displayColor = (1, 0.96, 0.82)
        }
    }

    over Mesh "StackProbe" {
        color3f inputs:displayColor = (0.96, 0.33, 0.7)
    }
}
`
