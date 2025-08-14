# 🚀 Guía Maestra para Crear TUI Profesionales al Estilo k9s

**Fecha de análisis:** 7 de agosto de 2025  
**Basado en:** Análisis exhaustivo del código fuente de k9s

---

## 📋 Contexto y Objetivos

Esta guía proporciona un blueprint completo para crear Terminal User Interfaces (TUI) profesionales usando los mismos patrones arquitectónicos y de diseño que k9s (Kubernetes CLI). La aplicación resultante será robusta, extensible y mantendrá una excelente experiencia de usuario.

## 🏗️ Arquitectura Principal

### 1. **Estructura Fundamental de Capas**

```
Aplicación TUI
├── 📱 Capa de UI (internal/ui/)
│   ├── App (aplicación principal)
│   ├── Pages (gestor de páginas/vistas)
│   ├── Componentes básicos (Table, Tree, Menu, Prompt)
│   └── Dialogs (modales y confirmaciones)
├── 🎭 Capa de Vista (internal/view/)
│   ├── Vistas específicas de recursos
│   ├── Gestión de comandos y navegación
│   └── Extenders de funcionalidad
├── 🎨 Capa de Renderizado (internal/render/)
│   ├── Renderizadores por tipo de recurso
│   ├── Sistema de colores dinámico
│   └── Formateo y presentación
├── ⚙️ Capa de Configuración (internal/config/)
│   ├── Skins y temas visuales
│   ├── Hotkeys y aliases
│   └── Configuración de contexto
└── 📊 Capa de Modelo (internal/model/)
    ├── Gestión de estado
    ├── Listeners y observadores
    └── Buffers de comando
```

### 2. **Componente App Principal**

Implementar una clase App que:

```go
type App struct {
    *tview.Application           // Biblioteca TUI base
    Configurator                 // Sistema de configuración
    Main         *Pages          // Gestor de páginas
    flash        *Flash          // Sistema de notificaciones
    actions      *KeyActions     // Gestión de atajos de teclado
    views        map[string]Primitive  // Registro de vistas
    cmdBuff      *FishBuff       // Buffer de comandos
    running      bool            // Estado de ejecución
}
```

**Funcionalidades clave:**
- Gestión del ciclo de vida de la aplicación
- Sistema de actualización asíncrona (`QueueUpdate`, `QueueUpdateDraw`)
- Manejo centralizado de eventos de teclado
- Configuración dinámica de estilos y temas

### 3. **Sistema de Páginas y Navegación**

```go
type Pages struct {
    *tview.Pages    // Control de páginas de tview
    *Stack          // Pila de navegación
}
```

**Características:**
- Stack de navegación para ir hacia atrás/adelante
- Soporte para diálogos modales superpuestos
- Transiciones suaves entre vistas
- Persistencia del estado de navegación

## 🎨 Sistema de Renderizado y Colores

### 1. **Coloreado Dinámico**

Implementar `ColorerFunc` que evalúa el estado de cada fila:

```go
type ColorerFunc func(namespace string, header Header, rowEvent *RowEvent) tcell.Color

// Ejemplo de implementación
func PodColorerFunc() ColorerFunc {
    return func(ns string, h Header, re *RowEvent) tcell.Color {
        statusIdx, _ := h.IndexOf("STATUS", true)
        status := strings.TrimSpace(re.Row.Fields[statusIdx])
        
        switch status {
        case "Running":
            return StdColor
        case "Pending":
            return PendingColor
        case "Error", "Failed":
            return ErrColor
        default:
            return DefaultColor
        }
    }
}
```

### 2. **Sistema de Colores Dinámico - Arquitectura de 3 Niveles**

k9s implementa un sofisticado sistema de colores que opera en tres niveles jerárquicos, permitiendo temas personalizables y colores dinámicos basados en el estado de los recursos.

#### **🎨 Nivel 1: Configuración YAML (Temas/Skins)**

Los colores se definen en archivos YAML de temas ubicados en `skins/`:

```yaml
# Ejemplo: skins/dracula.yaml
k9s:
  body:
    fgColor: "#f8f8f2"          # Color de texto principal
    bgColor: "#282a36"          # Color de fondo principal
    logoColor: "#bd93f9"        # Color del logo

  frame:
    border:
      fgColor: "#44475a"        # Color de bordes normales
      focusColor: "#6272a4"     # Color de bordes con foco
    
    status:
      newColor: "#50fa7b"       # Recursos nuevos (verde)
      modifyColor: "#ffb86c"    # Recursos modificados (naranja)
      addColor: "#8be9fd"       # Recursos añadidos (cyan)
      errorColor: "#ff5555"     # Recursos con error (rojo)
      highlightColor: "#f1fa8c" # Recursos destacados (amarillo)
      killColor: "#ff79c6"      # Recursos eliminados (magenta)
      completedColor: "#6272a4" # Recursos completados (gris)

  views:
    table:
      fgColor: "#f8f8f2"
      bgColor: "#282a36"
      cursorColor: "#bd93f9"    # Color del cursor/selección
      header:
        fgColor: "#50fa7b"      # Headers de tabla
        bgColor: "#282a36"
```

#### **⚙️ Nivel 2: Aplicación Global (tview.Styles)**

Los colores se aplican globalmente al framework tview en el método `Update()`:

```go
// internal/config/styles.go
func (s *Styles) Update() {
    // Aplicar colores base a tview
    tview.Styles.PrimitiveBackgroundColor = s.BgColor()
    tview.Styles.PrimaryTextColor = s.FgColor()
    tview.Styles.BorderColor = s.K9s.Frame.Border.FgColor.Color()
    tview.Styles.FocusColor = s.K9s.Frame.Border.FocusColor.Color()
    tview.Styles.TitleColor = s.FgColor()
    
    // Actualizar variables globales para renderizadores
    model1.ModColor = s.K9s.Frame.Status.ModifyColor.Color()
    model1.AddColor = s.K9s.Frame.Status.AddColor.Color()
    model1.PendingColor = s.K9s.Frame.Status.PendingColor.Color()
    model1.ErrColor = s.K9s.Frame.Status.ErrorColor.Color()
    model1.StdColor = s.FgColor()
    model1.HighlightColor = s.K9s.Frame.Status.HighlightColor.Color()
    model1.KillColor = s.K9s.Frame.Status.KillColor.Color()
    model1.CompletedColor = s.K9s.Frame.Status.CompletedColor.Color()
    
    // Notificar cambios a todos los componentes
    s.fireStylesChanged()
}
```

#### **🖌️ Nivel 3: Colores Dinámicos por Estado (ColorerFunc)**

Cada renderizador implementa una función `ColorerFunc` que determina el color de cada fila según su estado:

```go
// Ejemplo: internal/render/pod.go
func (*Pod) ColorerFunc() model1.ColorerFunc {
    return func(ns string, h model1.Header, re *model1.RowEvent) tcell.Color {
        // Color base por defecto
        c := model1.DefaultColorer(ns, h, re)
        
        // Buscar columna STATUS
        idx, ok := h.IndexOf("STATUS", true)
        if !ok {
            return c
        }
        
        // Aplicar color según el estado específico
        status := strings.TrimSpace(re.Row.Fields[idx])
        switch status {
        case "Pending", "ContainerCreating":
            return model1.PendingColor      // Amarillo/Naranja
        case "Running":
            if c != model1.ErrColor {
                return model1.StdColor      // Color normal
            }
        case "Terminating":
            return model1.KillColor         // Rojo/Magenta
        case "Completed":
            return model1.CompletedColor    // Gris
        case "Error", "Failed":
            return model1.ErrColor          // Rojo intenso
        }
        
        return c
    }
}
```

#### **🔧 Implementación en Tu TUI**

Para implementar este sistema en tu aplicación:

**1. Crear estructura de colores:**
```go
// internal/render/colorer.go
type EmailColorer struct{}

func (EmailColorer) ColorerFunc() func(email *Email, column string) tcell.Color {
    return func(email *Email, column string) tcell.Color {
        switch strings.ToUpper(column) {
        case "STATUS":
            if email.IsUnread {
                return UnreadColor    // Azul para no leído
            }
            return ReadColor          // Gris para leído
            
        case "FROM":
            if email.IsUnread {
                return tcell.ColorYellow  // Remitente destacado
            }
            return tcell.ColorWhite
            
        case "SUBJECT":
            if email.IsImportant {
                return ImportantColor     // Rojo para importante
            }
            if email.IsUnread {
                return tcell.ColorWhite   // Blanco brillante
            }
            return tcell.ColorGray        // Gris para leído
        }
        return tcell.ColorWhite
    }
}
```

**2. Aplicar colores en las vistas:**
```go
// En tu tabla/lista
func (el *EmailList) updateRow(email *Email, row int) {
    colorer := el.colorer.ColorerFunc()
    
    // Aplicar color a cada celda según su columna
    statusColor := colorer(email, "STATUS")
    fromColor := colorer(email, "FROM")
    subjectColor := colorer(email, "SUBJECT")
    
    el.SetCell(row, 0, tview.NewTableCell(status).SetTextColor(statusColor))
    el.SetCell(row, 1, tview.NewTableCell(from).SetTextColor(fromColor))
    el.SetCell(row, 2, tview.NewTableCell(subject).SetTextColor(subjectColor))
}
```

**3. Reaccionar a cambios de tema:**
```go
// Implementar StyleListener
func (el *EmailList) StylesChanged(s *config.Styles) {
    // Actualizar colores cuando cambia el tema
    el.updateColors()
    el.refreshTable()
}
```

**✅ Ventajas de esta arquitectura:**
- **Temas completamente personalizables**
- **Colores que reflejan el estado en tiempo real**
- **Cambio de tema sin reiniciar**
- **Coherencia visual en toda la aplicación**
- **Extensibilidad para nuevos tipos de contenido**

**Implementar:**
- Cargado dinámico de temas desde archivos YAML
- Validación con JSON Schema
- Sistema de listeners para cambios de estilo
- Variables globales de color sincronizadas
- ColorerFunc específicas por tipo de recurso

## ⌨️ Sistema de Eventos y Navegación

### 1. **Gestión de Teclas**

```go
type KeyActions struct {
    actions map[tcell.Key]KeyAction
    mx      sync.RWMutex
}

type KeyAction struct {
    Description string
    Action      ActionHandler
    Opts        ActionOpts  // visible, shared, plugin, etc.
}
```

**Patrones de implementación:**
- Mapeo jerárquico de teclas (global → vista → componente)
- Acciones compartidas vs específicas
- Sistema de hints dinámico para mostrar atajos disponibles
- Soporte para combinaciones de teclas (Ctrl+, Shift+, etc.)

### 2. **Patrón de Keyboard Handling**

```go
func (c *Component) keyboard(evt *tcell.EventKey) *tcell.EventKey {
    if action, ok := c.actions.Get(AsKey(evt)); ok {
        return action.Action(evt)
    }
    return evt  // Propagar evento si no se maneja
}
```

### 3. **Sistema de Comandos**

Implementar un prompt inteligente con:
- Autocompletado fuzzy
- Historial de comandos
- Sugerencias contextuales
- Validación en tiempo real
- Modo comando vs modo filtro

## 🔧 Sistema de Configuración

### 1. **Hotkeys Personalizables**

```yaml
hotKeys:
  shift-0:
    shortCut: "Shift-0"
    description: "View Workloads"
    command: "workload"
    keepHistory: true
```

### 2. **Aliases de Comandos**

```yaml
aliases:
  po: "v1/pods"
  svc: "v1/services"
  deploy: "apps/v1/deployments"
```

### 3. **Configuración por Contexto**

- Archivos de configuración separados por entorno
- Merge automático de configuraciones globales y específicas
- Validación con JSON Schema
- Recarga en caliente sin reiniciar

## 📊 Componentes UI Esenciales

### 1. **Tabla Inteligente**

```go
type Table struct {
    *SelectTable
    gvr          *GVR           // Identificador del recurso
    sortCol      SortColumn     // Columna de ordenamiento
    actions      *KeyActions    // Acciones específicas
    cmdBuff      *FishBuff      // Buffer de filtros
    colorerFn    ColorerFunc    // Función de coloreado
    decorateFn   DecorateFunc   // Decorador de filas
}
```

**Características:**
- Ordenamiento multi-columna
- Filtrado fuzzy en tiempo real
- Selección múltiple con marcadores
- Paginación automática
- Coloreado dinámico por estado
- Acciones contextuales por fila

### 2. **Sistema de Diálogos**

Tipos de diálogos necesarios:
- **Confirmación:** Para acciones destructivas
- **Error:** Con mascota ASCII (vaca) para hacer los errores amigables
- **Prompt:** Para entrada de datos
- **Selección:** Listas de opciones
- **Transferencia:** Barras de progreso

### 3. **Menu de Navegación**

Menu dinámico que muestra:
- Atajos disponibles por contexto
- Descripción de acciones
- Agrupación inteligente por categorías
- Resaltado de acciones peligrosas

## 🔄 Gestión de Estado y Modelos

### 1. **Patrón Observer**

```go
type TableListener interface {
    TableDataChanged(*TableData)
    TableLoadFailed(error)
    TableNoData(*TableData)
}
```

### 2. **Buffers de Comando**

```go
type CmdBuff struct {
    buff       []rune
    suggestion string
    listeners  map[BuffWatcher]struct{}
    kind       BufferKind  // Command vs Filter
    active     bool
}
```

### 3. **Gestión de Focus**

Sistema para manejar el focus entre componentes:
- Stack de focus para navegación con Tab
- Restauración automática al volver de diálogos
- Indicadores visuales claros de focus
- Manejo de focus en layouts complejos

## 🎯 Patrones de Implementación Específicos

### 1. **Factory Pattern para Vistas**

```go
type MetaViewer struct {
    viewerFn func(*GVR) ResourceViewer
}

type MetaViewers map[*GVR]MetaViewer

func loadCustomViewers() MetaViewers {
    m := make(MetaViewers, 30)
    m[PodGVR] = MetaViewer{viewerFn: NewPod}
    m[ServiceGVR] = MetaViewer{viewerFn: NewService}
    // ... más vistas
    return m
}
```

### 2. **Sistema de Extensions**

```go
type ActionExtender interface {
    BindKeys(ResourceViewer)
}

// Extenders específicos:
// - LogsExtender: Para ver logs
// - PortForwardExtender: Para port forwarding  
// - ScaleExtender: Para escalar recursos
// - RestartExtender: Para reiniciar
```

### 3. **Renderizado Modular**

```go
type Renderer interface {
    Header(namespace string) Header
    Render(object any, namespace string, row *Row) error
    ColorerFunc() ColorerFunc
}
```

## 🚀 Recomendaciones de Implementación

### 1. **Tecnologías Base**
- **Go + tview:** Para la base TUI (como k9s)
- **tcell:** Para manejo low-level de terminal
- **YAML:** Para configuración
- **JSON Schema:** Para validación

## 📦 Módulos de Go Utilizados por k9s

### **Bibliotecas TUI Core**

```go
// Interfaz de usuario terminal principal
github.com/derailed/tview v0.8.5        // Framework TUI principal (fork de rivo/tview)
github.com/derailed/tcell/v2 v2.3.1-rc.4 // Control de terminal low-level (fork de gdamore/tcell)

// Utilidades de terminal
github.com/mattn/go-runewidth v0.0.16   // Manejo de ancho de caracteres Unicode
github.com/mattn/go-colorable v0.1.14   // Soporte de color cross-platform
github.com/atotto/clipboard v0.1.4      // Acceso al portapapeles del sistema
```

### **Configuración y Archivos**

```go
// Configuración YAML/JSON
gopkg.in/yaml.v3 v3.0.1                 // Parser/serializer YAML
github.com/xeipuuv/gojsonschema v1.2.0   // Validación JSON Schema

// Archivos y sistema
github.com/adrg/xdg v0.5.3               // Directorios estándar XDG
github.com/fsnotify/fsnotify v1.9.0      // Watch de archivos del sistema
```

### **Búsqueda y Filtrado**

```go
// Búsqueda fuzzy
github.com/sahilm/fuzzy v0.1.1           // Algoritmo fuzzy matching para filtros

// Ordenamiento
github.com/fvbommel/sortorder v1.1.0     // Ordenamiento natural (ej: file1, file10, file2)
```

### **Colores y Styling**

```go
// Sistema de colores
github.com/fatih/color v1.18.0           // Colores ANSI en terminal
github.com/lmittmann/tint v1.0.7         // Colores para logging estructurado
```

### **CLI y Comandos**

```go
// Framework CLI
github.com/spf13/cobra v1.9.1            // Framework de línea de comandos

// Expresiones y queries
github.com/itchyny/gojq v0.12.17         // Implementación de jq en Go
```

### **Renderizado de Tablas**

```go
// Renderizado de datos tabulares
github.com/olekukonko/tablewriter v1.0.8 // Generación de tablas ASCII
```

### **Utilidades de Sistema**

```go
// Manejo de errores
github.com/go-errors/errors v1.5.1       // Stack traces mejorados

// Backoff y retry
github.com/cenkalti/backoff/v4 v4.3.0    // Algoritmos de backoff exponencial

// Texto y Unicode
golang.org/x/text v0.27.0                // Soporte completo de Unicode
```

### **Ejemplo de Imports Típicos en k9s**

```go
// Archivo típico de UI component
package ui

import (
    "context"
    "sync"
    
    "github.com/derailed/k9s/internal/config"
    "github.com/derailed/k9s/internal/model"
    "github.com/derailed/tcell/v2"        // Control de eventos de teclado
    "github.com/derailed/tview"           // Componentes TUI
)

// Archivo típico de vista
package view

import (
    "github.com/derailed/k9s/internal/ui"
    "github.com/derailed/k9s/internal/client"
    "github.com/derailed/k9s/internal/render"
    "github.com/sahilm/fuzzy"             // Búsqueda fuzzy
    "gopkg.in/yaml.v3"                    // Configuración YAML
)
```

### **Dependencias Clave para Replicar**

Para crear una TUI similar, necesitarás estos módulos esenciales:

```bash
# Módulos TUI principales
go get github.com/rivo/tview            # (o usar el fork derailed/tview)
go get github.com/gdamore/tcell/v2      # (o usar el fork derailed/tcell)

# Configuración y archivos
go get gopkg.in/yaml.v3
go get github.com/xeipuuv/gojsonschema
go get github.com/fsnotify/fsnotify

# Búsqueda y filtrado
go get github.com/sahilm/fuzzy
go get github.com/fvbommel/sortorder

# Colores y CLI
go get github.com/fatih/color
go get github.com/spf13/cobra

# Utilidades
go get github.com/atotto/clipboard
go get github.com/mattn/go-runewidth
go get github.com/cenkalti/backoff/v4
```

### **Notas Importantes sobre los Forks**

k9s utiliza **forks personalizados** de las bibliotecas principales:

1. **`github.com/derailed/tview`** (fork de `rivo/tview`)
   - Añade funcionalidades específicas para k9s
   - Mejoras en el manejo de eventos
   - Componentes personalizados

2. **`github.com/derailed/tcell/v2`** (fork de `gdamore/tcell`)
   - Optimizaciones de rendimiento
   - Soporte mejorado para diferentes terminales
   - Manejo específico de eventos de teclado

Estos forks proporcionan funcionalidades adicionales que no están en las versiones originales, por lo que para replicar completamente la funcionalidad de k9s, podrías necesitar usar los forks o implementar funcionalidades similares.

### 2. **Estructura de Proyecto**

```
proyecto/
├── cmd/                 # Comando principal
├── internal/
│   ├── ui/             # Componentes UI básicos
│   ├── view/           # Vistas específicas  
│   ├── render/         # Sistema de renderizado
│   ├── config/         # Configuración
│   ├── model/          # Modelos y estado
│   └── theme/          # Gestión de temas
├── assets/             # Recursos estáticos
├── skins/              # Temas predefinidos
└── configs/            # Configuraciones base
```

### 3. **Testing Strategy**
- Tests unitarios para cada componente
- Tests de integración para flujos completos
- Tests de UI con simulación de eventos
- Benchmarks para rendimiento de renderizado

### 4. **Performance Considerations**
- Lazy loading de datos grandes
- Virtualización de listas largas
- Debouncing para filtros en tiempo real
- Pool de objetos para reducir GC pressure
- Renderizado incremental

## 🎨 Elementos de UX Avanzados

### 1. **Feedback Visual**
- Spinners para operaciones largas
- Barras de progreso para transferencias
- Flasheos para notificaciones
- Colores semánticos (verde=ok, rojo=error, amarillo=warning)

### 2. **Navegación Intuitiva**
- Breadcrumbs para mostrar contexto
- Historial de navegación (adelante/atrás)
- Shortcuts para vistas frecuentes
- Status bar informativo

### 3. **Accesibilidad**
- Soporte completo de teclado
- Indicadores visuales claros
- Mensajes de error descriptivos
- Hints contextuales

## 🔍 Detalles de Implementación k9s

### Componentes Clave Analizados

#### 1. **Aplicación Principal (`internal/ui/app.go`)**
- Gestión centralizada de eventos de teclado
- Sistema de vistas registradas con mapa
- Buffer de comandos con listener pattern
- Configurador de estilos integrado

#### 2. **Sistema de Páginas (`internal/ui/pages.go`)**
- Integración con tview.Pages
- Stack para navegación hacia atrás
- Detección automática de diálogos
- Gestión de componentes activos

#### 3. **Tabla Avanzada (`internal/ui/table.go`)**
- Selección múltiple con marcadores
- Filtrado fuzzy en tiempo real
- Ordenamiento dinámico
- Coloreado contextual por estado

#### 4. **Sistema de Renderizado (`internal/render/`)**
- Renderizadores específicos por recurso (Pod, Service, etc.)
- Funciones de coloreado personalizables
- Headers dinámicos con metadatos
- Formateo inteligente de datos

#### 5. **Configuración Dinámica (`internal/config/`)**
- Carga de skins desde YAML
- Hotkeys personalizables
- Aliases de comandos
- Validación con JSON Schema

#### 6. **Gestión de Eventos (`internal/ui/key.go`)**
- Mapeo completo de teclas (estándar, shift, ctrl)
- Acciones jerárquicas con propagación
- Sistema de hints contextual
- Soporte para combinaciones complejas

## 📋 Checklist de Implementación

### Fase 1: Base
- [ ] Configurar proyecto Go con tview/tcell
- [ ] Implementar App principal con Pages
- [ ] Sistema básico de eventos de teclado
- [ ] Configuración YAML básica

### Fase 2: Componentes Core
- [ ] Tabla con filtrado y selección
- [ ] Sistema de diálogos modales
- [ ] Prompt con autocompletado
- [ ] Menu dinámico de acciones

### Fase 3: Avanzado
- [ ] Sistema de skins/temas
- [ ] Hotkeys personalizables
- [ ] Sistema de renderizado modular
- [ ] Gestión de estado con observers

### Fase 4: Polish
- [ ] Testing exhaustivo
- [ ] Performance optimization
- [ ] Documentación de usuario
- [ ] Temas predefinidos

## 🎓 Lecciones Clave de k9s

1. **Modularidad:** Cada componente tiene responsabilidades claras
2. **Configurabilidad:** Todo es personalizable sin recompilación
3. **Extensibilidad:** Sistema de plugins y extenders
4. **Performance:** Actualizaciones asíncronas y renderizado eficiente
5. **UX:** Feedback visual constante y navegación intuitiva

---

## 🎯 Ejemplo Práctico: Gmail TUI

### Implementación Real del Sistema de Colores

Para demostrar cómo aplicar estos conceptos, aquí está la implementación real del sistema de colores en nuestro Gmail TUI:

#### **1. Configuración de Tema** (`skins/gmail-dark.yaml`):
```yaml
gmailTUI:
  body:
    fgColor: "#f8f8f2"
    bgColor: "#282a36"
    logoColor: "#bd93f9"
  
  frame:
    border:
      fgColor: "#44475a"
      focusColor: "#6272a4"
    
    title:
      fgColor: "#f8f8f2"
      bgColor: "#282a36"
      highlightColor: "#f1fa8c"
      counterColor: "#50fa7b"
      filterColor: "#8be9fd"

  views:
    table:
      fgColor: "#f8f8f2"
      bgColor: "#282a36"
      headerFgColor: "#50fa7b"
      headerBgColor: "#282a36"

  # Colores específicos para emails
  email:
    unreadColor: "#ffb86c"      # Naranja para no leídos
    readColor: "#6272a4"        # Gris para leídos
    importantColor: "#ff5555"   # Rojo para importantes
    sentColor: "#50fa7b"        # Verde para enviados
    draftColor: "#f1fa8c"       # Amarillo para borradores
```

#### **2. Renderizador de Emails** (`internal/render/email.go`):
```go
package render

import (
    "strings"
    "github.com/gdamore/tcell/v2"
    "github.com/yourusername/gmail-tui/internal/gmail"
)

// EmailColorer maneja los colores de emails
type EmailColorer struct {
    UnreadColor    tcell.Color
    ReadColor      tcell.Color
    ImportantColor tcell.Color
    SentColor      tcell.Color
    DraftColor     tcell.Color
}

// ColorerFunc devuelve función de coloreo para emails
func (ec *EmailColorer) ColorerFunc() func(*gmail.Email, string) tcell.Color {
    return func(email *gmail.Email, column string) tcell.Color {
        switch strings.ToUpper(column) {
        case "STATUS":
            if email.IsUnread {
                return ec.UnreadColor  // 🔵 Azul para no leído
            }
            return ec.ReadColor        // ⚪ Gris para leído
            
        case "FROM":
            if email.IsImportant {
                return ec.ImportantColor  // 🔴 Rojo para importante
            }
            if email.IsUnread {
                return ec.UnreadColor     // 🟠 Naranja para no leído
            }
            return tcell.ColorWhite
            
        case "SUBJECT":
            if contains(email.Labels, "DRAFT") {
                return ec.DraftColor      // 🟡 Amarillo para borrador
            }
            if contains(email.Labels, "SENT") {
                return ec.SentColor       // 🟢 Verde para enviado
            }
            if email.IsUnread {
                return tcell.ColorWhite   // ⚪ Blanco brillante
            }
            return ec.ReadColor           // ⚫ Gris para leído
        }
        return tcell.ColorWhite
    }
}

// UpdateFromStyles actualiza colores desde configuración
func (ec *EmailColorer) UpdateFromStyles(styles *config.Styles) {
    ec.UnreadColor = styles.Email.UnreadColor.Color()
    ec.ReadColor = styles.Email.ReadColor.Color()
    ec.ImportantColor = styles.Email.ImportantColor.Color()
    ec.SentColor = styles.Email.SentColor.Color()
    ec.DraftColor = styles.Email.DraftColor.Color()
}
```

#### **3. Aplicación en EmailList** (`internal/view/email_list.go`):
```go
// updateTable aplica colores dinámicos a cada fila
func (el *EmailList) updateTable() {
    // Limpiar filas existentes
    for i := el.GetRowCount() - 1; i > 0; i-- {
        el.RemoveRow(i)
    }

    colorer := el.colorer.ColorerFunc()
    
    for i, email := range el.emails {
        row := i + 1
        
        // Aplicar colores específicos por columna
        statusColor := colorer(email, "STATUS")
        fromColor := colorer(email, "FROM")
        subjectColor := colorer(email, "SUBJECT")
        dateColor := tcell.ColorGray
        
        // Crear celdas con colores apropiados
        statusText := " "
        if email.IsUnread {
            statusText = "●"
        }
        
        el.SetCell(row, 0, tview.NewTableCell(statusText).
            SetTextColor(statusColor).
            SetAlign(tview.AlignCenter))
            
        el.SetCell(row, 1, tview.NewTableCell(el.truncateString(email.From, 25)).
            SetTextColor(fromColor))
            
        el.SetCell(row, 2, tview.NewTableCell(el.truncateString(email.Subject, 50)).
            SetTextColor(subjectColor))
            
        el.SetCell(row, 3, tview.NewTableCell(el.formatDate(email.Date)).
            SetTextColor(dateColor).
            SetAlign(tview.AlignRight))
    }
}

// StylesChanged reacciona a cambios de tema
func (el *EmailList) StylesChanged(s *config.Styles) {
    // Actualizar colores del renderizador
    el.colorer.UpdateFromStyles(s)
    
    // Refrescar toda la tabla
    el.app.QueueUpdateDraw(func() {
        el.updateTable()
    })
}
```

#### **4. Resultado Visual:**

```
┌─ Emails ────────────────────────────────────────────────┐
│ Status │ From              │ Subject                     │
├────────┼───────────────────┼─────────────────────────────┤
│   🔵●  │ John Doe          │ Meeting tomorrow            │ <- No leído (naranja)
│   ⚪   │ Jane Smith        │ Project update              │ <- Leído (gris)
│   🔴●  │ Boss              │ URGENT: Review needed       │ <- Importante (rojo)
│   🟢   │ Me                │ Re: Vacation request        │ <- Enviado (verde)
│   🟡   │ Draft             │ Email draft                 │ <- Borrador (amarillo)
└─────────────────────────────────────────────────────────┘
```

### Beneficios Reales

✅ **Información visual instantánea** - Los usuarios ven el estado sin leer texto  
✅ **Consistencia** - Mismos colores en toda la aplicación  
✅ **Personalización** - Cada usuario puede tener su tema preferido  
✅ **Accesibilidad** - Soporte para diferentes esquemas de color  
✅ **Escalabilidad** - Fácil añadir nuevos estados y colores

---

Esta guía captura la esencia de la arquitectura TUI de k9s y proporciona un roadmap detallado para implementar interfaces similares en otros proyectos. La clave está en la modularidad, la configurabilidad y la atención al detalle en la experiencia de usuario.

**¡Éxito construyendo tu TUI profesional con sistema de colores dinámico!** 🚀
