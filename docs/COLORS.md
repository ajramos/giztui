# Sistema de Colores de Gmail TUI

Gmail TUI implementa un sistema de colores dinámico inspirado en k9s que permite personalizar completamente la apariencia visual de la aplicación.

## 🎨 Arquitectura del Sistema de Colores

### Niveles de Configuración

1. **Archivos YAML de Temas** (`skins/`)
   - Definición de colores en formato YAML
   - Temas predefinidos (dark, light)
   - Personalización completa

2. **Configuración de Aplicación**
   - Carga dinámica de temas
   - Aplicación global de colores
   - Actualización en tiempo real

3. **Renderizadores Específicos**
   - Colores dinámicos por estado de email
   - Funciones de coloreado personalizables
   - Lógica de estado integrada

## 📁 Estructura de Archivos

```
gmail-tui/
├── skins/
│   ├── gmail-dark.yaml     # Tema oscuro (Dracula)
│   └── gmail-light.yaml    # Tema claro
├── internal/
│   ├── config/
│   │   └── colors.go       # Sistema de colores base
│   └── render/
│       └── email.go        # Renderizador de emails
└── docs/
    └── COLORS.md           # Esta documentación
```

## 🎯 Colores por Estado de Email

### Estados Principales

| Estado | Color | Descripción |
|--------|-------|-------------|
| **No Leído** | `#ffb86c` (Naranja) | Emails nuevos sin leer |
| **Leído** | `#6272a4` (Gris) | Emails ya leídos |
| **Importante** | `#ff5555` (Rojo) | Emails marcados como importantes |
| **Enviado** | `#50fa7b` (Verde) | Emails enviados por el usuario |
| **Borrador** | `#f1fa8c` (Amarillo) | Borradores guardados |

### Estados Secundarios

| Estado | Color | Descripción |
|--------|-------|-------------|
| **Remitente (No Leído)** | `#ffb86c` | Nombre del remitente destacado |
| **Remitente (Importante)** | `#ff5555` | Remitente de email importante |
| **Asunto (No Leído)** | `#ffffff` | Asunto en blanco brillante |
| **Asunto (Leído)** | `#6272a4` | Asunto en gris |

## 📝 Formato de Archivos de Tema

### Estructura YAML

```yaml
gmailTUI:
  body:
    fgColor: "#f8f8f2"          # Texto principal
    bgColor: "#282a36"          # Fondo principal
    logoColor: "#bd93f9"        # Logo

  frame:
    border:
      fgColor: "#44475a"        # Bordes normales
      focusColor: "#6272a4"     # Bordes con foco
    
    title:
      fgColor: "#f8f8f2"        # Título
      bgColor: "#282a36"        # Fondo del título
      highlightColor: "#f1fa8c" # Resaltado
      counterColor: "#50fa7b"   # Contador
      filterColor: "#8be9fd"    # Filtro

  table:
    fgColor: "#f8f8f2"          # Texto de tabla
    bgColor: "#282a36"          # Fondo de tabla
    headerFgColor: "#50fa7b"    # Headers
    headerBgColor: "#282a36"    # Fondo de headers

  email:
    unreadColor: "#ffb86c"      # No leídos
    readColor: "#6272a4"        # Leídos
    importantColor: "#ff5555"   # Importantes
    sentColor: "#50fa7b"        # Enviados
    draftColor: "#f1fa8c"       # Borradores
```

### Formatos de Color Soportados

- **Hexadecimal**: `#ff5555`
- **Nombres de color**: `red`, `blue`, `green`
- **ANSI**: `1`, `2`, `3` (códigos ANSI)
- **Default**: `default` (color por defecto del terminal)

## 🔧 Implementación Técnica

### Renderizador de Emails

```go
// EmailColorer maneja los colores de emails
type EmailColorer struct {
    UnreadColor    tcell.Color
    ReadColor      tcell.Color
    ImportantColor tcell.Color
    SentColor      tcell.Color
    DraftColor     tcell.Color
}

// ColorerFunc devuelve función de coloreo para emails
func (ec *EmailColorer) ColorerFunc() func(*googleGmail.Message, string) tcell.Color {
    return func(message *googleGmail.Message, column string) tcell.Color {
        switch strings.ToUpper(column) {
        case "STATUS":
            if ec.isUnread(message) {
                return ec.UnreadColor  // 🔵 Azul para no leído
            }
            return ec.ReadColor        // ⚪ Gris para leído
            
        case "FROM":
            if ec.isImportant(message) {
                return ec.ImportantColor  // 🔴 Rojo para importante
            }
            if ec.isUnread(message) {
                return ec.UnreadColor     // 🟠 Naranja para no leído
            }
            return tcell.ColorWhite
            
        case "SUBJECT":
            if ec.isDraft(message) {
                return ec.DraftColor      // 🟡 Amarillo para borrador
            }
            if ec.isSent(message) {
                return ec.SentColor       // 🟢 Verde para enviado
            }
            if ec.isUnread(message) {
                return tcell.ColorWhite   // ⚪ Blanco brillante
            }
            return ec.ReadColor           // ⚫ Gris para leído
        }
        return tcell.ColorWhite
    }
}
```

### Detección de Estados

```go
// Helper methods to determine email state
func (ec *EmailColorer) isUnread(message *googleGmail.Message) bool {
    // Check if message has UNREAD label
    for _, labelId := range message.LabelIds {
        if labelId == "UNREAD" {
            return true
        }
    }
    return false
}

func (ec *EmailColorer) isImportant(message *googleGmail.Message) bool {
    // Check for important labels
    importantLabels := []string{"IMPORTANT", "PRIORITY", "URGENT"}
    for _, labelId := range message.LabelIds {
        for _, important := range importantLabels {
            if strings.Contains(strings.ToUpper(labelId), important) {
                return true
            }
        }
    }
    return false
}
```

## 🎨 Temas Predefinidos

### Tema Oscuro (Dracula)

Basado en la paleta de colores Dracula, proporciona una experiencia visual cómoda para uso nocturno.

**Características:**
- Fondo oscuro (`#282a36`)
- Texto claro (`#f8f8f2`)
- Acentos en púrpura (`#bd93f9`)
- Colores semánticos para estados

### Tema Claro

Diseñado para uso diurno y entornos con mucha luz.

**Características:**
- Fondo claro (`#ecf0f1`)
- Texto oscuro (`#2c3e50`)
- Acentos en azul (`#3498db`)
- Contraste optimizado

## 🚀 Uso Avanzado

### Crear un Tema Personalizado

1. **Copiar un tema existente**:
   ```bash
   cp skins/gmail-dark.yaml skins/my-theme.yaml
   ```

2. **Modificar colores**:
   ```yaml
   gmailTUI:
     email:
       unreadColor: "#ff6b6b"      # Tu color personalizado
       readColor: "#4ecdc4"        # Otro color
   ```

3. **Aplicar el tema**:
   ```go
   // En tu código
   colors := config.LoadColorsFromFile("skins/my-theme.yaml")
   app.emailRenderer.UpdateFromConfig(colors)
   ```

### Colores Dinámicos

Los colores se aplican dinámicamente según el estado del email:

- **No leído**: Naranja brillante
- **Importante**: Rojo de advertencia
- **Borrador**: Amarillo de atención
- **Enviado**: Verde de confirmación
- **Leído**: Gris discreto

## 🔍 Beneficios del Sistema

### Para Usuarios

✅ **Información visual instantánea** - Estados claros sin leer texto  
✅ **Personalización completa** - Temas adaptados a preferencias  
✅ **Accesibilidad mejorada** - Contraste optimizado  
✅ **Experiencia consistente** - Mismos colores en toda la app  

### Para Desarrolladores

✅ **Arquitectura modular** - Fácil extensión  
✅ **Configuración externa** - Sin recompilación  
✅ **Reutilización de código** - Patrones establecidos  
✅ **Testing simplificado** - Colores predecibles  

## 📋 Próximas Mejoras

- [ ] **Temas automáticos** - Detección de preferencia del sistema
- [ ] **Transiciones suaves** - Animaciones entre temas
- [ ] **Paletas personalizadas** - Generador de temas
- [ ] **Exportación/Importación** - Compartir temas
- [ ] **Modo alto contraste** - Accesibilidad avanzada

---

**¡El sistema de colores de Gmail TUI proporciona una experiencia visual rica y personalizable!** 🎨

