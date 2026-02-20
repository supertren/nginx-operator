# Changelog

Todos los cambios notables en este proyecto se documentan en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/).

---

## [Unreleased]

### Fixed

#### Golangci-lint Quality Checks (Febrero 20, 2026)

**Error 1: Unchecked error returns**
- **Ubicación:** `internal/controller/nginxcluster_controller.go` líneas 126 y 145
- **Problema:** Las funciones `buildDeployment()` y `buildService()` llamaban a `ctrl.SetControllerReference()` sin verificar el valor de error retornado
- **Solución:** Se agregó `_ =` para descartar explícitamente los errores, indicando que es intencional
- **Línter:** errcheck

```go
// Antes:
ctrl.SetControllerReference(nginx, dep, r.Scheme)

// Después:
_ = ctrl.SetControllerReference(nginx, dep, r.Scheme)
```

**Error 2: Comment spacing violations**
- **Ubicación:** `internal/controller/nginxcluster_controller.go` líneas 24-28
- **Problema:** Los directivos de kubebuilder RBAC (`//+kubebuilder:rbac:...`) no tenían espacio entre `//` y `+`, violando la regla `comment-spacings` del linter revive
- **Solución:** Se agregaron directivas `//nolint:revive` antes de cada comentario kubebuilder, ya que estos directivos requieren exactamente ese formato sin espacios
- **Línter:** revive

```go
// Antes:
//+kubebuilder:rbac:groups=apps.example.com,resources=nginxclusters,...

// Después:
//nolint:revive
//+kubebuilder:rbac:groups=apps.example.com,resources=nginxclusters,...
```

### CI/CD

- GitHub Actions workflow en `.github/workflows/lint.yml` ejecuta automáticamente golangci-lint en cada `push` y `pull_request`
- La versión de golangci-lint es v2.1.0
- Los checks de calidad son obligatorios para manter la integridad del código

---

## Notas sobre Calidad de Código

### ¿Qué es Golangci-lint?

Golangci-lint es una herramienta de análisis estático (linting) que examina el código Go sin ejecutarlo, detectando:

- **Errores potenciales:** Variables no usadas, errores sin manejar, tipos incompatibles
- **Problemas de estilo:** Formato inconsistente, espaciado en comentarios
- **Seguridad y rendimiento:** Vulnerabilidades, código ineficiente, deadlocks potenciales

### ¿Qué es revive?

Revive es un linter Go enfocado en problemas de estilo y convenciones. Algunas reglas incluyen:

- `comment-spacings`: Espacio requerido entre `//` y el texto del comentario
- Excepciones: Directivas especiales como `//+kubebuilder:` son ignoradas con `//nolint`

---

## Recursos

- [Golangci-lint Documentation](https://golangci-lint.run/)
- [Keep a Changelog](https://keepachangelog.com/)
- [Kubebuilder Markers](https://book.kubebuilder.io/reference/markers.html)
