# Propuesta: Sistema de Ingesta de Emails a Obsidian

## Resumen Ejecutivo

Implementar un sistema integrado en Gmail TUI que permita reenviar emails seleccionados directamente a Obsidian en la carpeta `00-Inbox`, siguiendo los principios de arquitectura orientada a servicios del proyecto. El sistema facilitará el flujo de trabajo de segundo cerebro, permitiendo procesar emails importantes más tarde en un entorno de notas estructurado.

## Objetivos del Sistema

### **Objetivos Principales**
- **Ingesta Eficiente**: Reenviar emails a Obsidian con un atajo de tecla
- **Organización Automática**: Todos los emails van a `00-Inbox` para procesamiento posterior
- **Formato Estructurado**: Emails formateados en Markdown con metadatos completos
- **Prevención de Duplicados**: Sistema de historial para evitar reingesta
- **Integración Nativa**: Se siente como parte del flujo de trabajo de Obsidian

### **Objetivos Secundarios**
- **Templates Personalizables**: Diferentes formatos según el tipo de email
- **Historial Completo**: Tracking de todos los envíos con metadatos
- **Configuración Flexible**: Adaptable a diferentes workflows de Obsidian
- **Performance**: Procesamiento asíncrono sin bloquear la UI

## Arquitectura del Sistema

### **1. Capa de Servicios**

#### **Nuevo Servicio: ObsidianService**
```go
// internal/services/obsidian_service.go
type ObsidianService interface {
    IngestEmailToObsidian(ctx context.Context, message *gmail.Message, options ObsidianOptions) error
    FormatEmailForObsidian(message *gmail.Message, template ObsidianTemplate) (string, error)
    GetObsidianTemplates(ctx context.Context) ([]*ObsidianTemplate, error)
    ValidateObsidianConnection(ctx context.Context) error
    GetObsidianVaultPath() string
}

type ObsidianOptions struct {
    VaultPath        string
    TemplateName     string
    AccountEmail     string
    IncludeAttachments bool
    CustomMetadata   map[string]interface{}
}
```

#### **Servicio de Historial: ObsidianHistoryService**
```go
type ObsidianHistoryService interface {
    RecordForward(ctx context.Context, record ObsidianForwardRecord) error
    GetForwardHistory(ctx context.Context, messageID string) (*ObsidianForwardRecord, error)
    CheckIfAlreadyForwarded(ctx context.Context, messageID, accountEmail string) (bool, error)
    ListRecentForwards(ctx context.Context, limit int) ([]*ObsidianForwardRecord, error)
    UpdateForwardStatus(ctx context.Context, id int, status, errorMessage string) error
}

type ObsidianForwardRecord struct {
    ID            int
    MessageID     string
    AccountEmail  string
    ObsidianPath  string
    TemplateUsed  string
    ForwardDate   time.Time
    Status        string
    ErrorMessage  string
    FileSize      int64
    Metadata      map[string]interface{}
}
```

### **2. Integración en UI**

#### **Nuevo Atajo de Tecla: `Shift+O`**
```go
// internal/tui/keys.go
case 'O': // Shift+O
    if a.currentFocus == "search" {
        return nil
    }
    go a.sendEmailToObsidian()
    return nil
```

#### **Nuevo Archivo: `internal/tui/obsidian.go`**
- Funciones para manejo de UI de Obsidian
- Modal de configuración de ingesta
- Preview del contenido formateado
- Gestión de templates

## Estructura de Archivos en Obsidian

### **Organización del Sistema de Segundo Cerebro**
```
ObsidianVault/
├── 00-Inbox/                    # 📥 Punto de entrada único para ingesta
│   ├── 2024-01-15_Meeting_Summary_company.com.md
│   ├── 2024-01-16_Project_Update_team.org.md
│   └── 2024-01-17_Newsletter_Cloud_Updates_aws.com.md
├── Templates/                    # 📋 Plantillas del sistema
│   └── email_templates/
│       ├── standard.md
│       ├── meeting.md
│       └── project.md
└── 01-Processing/               # 🔄 Tu flujo de procesamiento
```

### **Convención de Nombres**
```go
// Formato: YYYY-MM-DD_Subject_FromDomain.md
func generateInboxFilename(message *gmail.Message) string {
    date := time.Now().Format("2006-01-02")
    subject := sanitizeFilename(message.Subject)
    from := extractDomain(message.From)
    
    return fmt.Sprintf("%s_%s_%s.md", date, subject, from)
}
```

## Sistema de Templates

### **1. Templates en Obsidian (Recomendado)**

#### **Ubicación**
- **Ruta**: `ObsidianVault/Templates/email_templates/`
- **Formato**: Archivos Markdown con variables de sustitución
- **Gestión**: Edición directa en Obsidian

#### **Template Estándar**
```markdown
---
title: "{{subject}}"
date: {{date}}
from: {{from}}
type: email
status: inbox
labels: {{labels}}
message_id: {{message_id}}
---

# {{subject}}

**From:** {{from}}  
**Date:** {{date}}  
**Labels:** {{labels}}

---

{{body}}

---

*Ingested from Gmail on {{ingest_date}}*
```

#### **Template para Reuniones**
```markdown
---
title: "{{subject}}"
date: {{date}}
from: {{from}}
type: meeting
status: inbox
tags: [meeting, action-items]
---

# {{subject}}

**Meeting Details:**
- **From:** {{from}}
- **Date:** {{date}}
- **Type:** Meeting/Follow-up

**Action Items:**
- [ ] 

**Notes:**
{{body}}

---

*Ingested from Gmail on {{ingest_date}}*
```

### **2. Templates Locales (Fallback)**

#### **Ubicación**
- **Ruta**: `internal/templates/email_templates.go`
- **Propósito**: Fallback cuando Obsidian no esté disponible
- **Gestión**: Código fuente de la aplicación

## Base de Datos y Persistencia

### **Nueva Tabla: obsidian_forward_history**
```sql
-- scripts/create_obsidian_history.sql
CREATE TABLE IF NOT EXISTS obsidian_forward_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
    account_email TEXT NOT NULL,
    obsidian_path TEXT NOT NULL,
    template_used TEXT,
    forward_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'success', -- success, failed, pending
    error_message TEXT,
    file_size INTEGER,
    metadata TEXT -- JSON con metadatos adicionales
);

-- Índices para búsquedas eficientes
CREATE INDEX IF NOT EXISTS idx_obsidian_history_message_id ON obsidian_forward_history(message_id);
CREATE INDEX IF NOT EXISTS idx_obsidian_history_date ON obsidian_forward_history(forward_date);
CREATE INDEX IF NOT EXISTS idx_obsidian_history_status ON obsidian_forward_history(status);
```

### **Metadatos Almacenados**
- **Identificación**: MessageID, AccountEmail
- **Ruta**: ObsidianPath, TemplateUsed
- **Estado**: Status, ErrorMessage, ForwardDate
- **Tamaño**: FileSize
- **Contexto**: Metadata (JSON con subject, from, date, labels)

## Flujo de Trabajo de Ingesta

### **1. Selección del Email**
- Usuario selecciona email en la lista
- Presiona `Shift+O` para iniciar ingesta

### **2. Verificación de Duplicados**
- Sistema verifica si el email ya fue ingerido
- Previene reingesta del mismo contenido

### **3. Aplicación de Template**
- Selección automática o manual del template
- Sustitución de variables: {{subject}}, {{from}}, {{body}}, etc.
- Preview del contenido formateado

### **4. Creación en Obsidian**
- Generación de nombre de archivo único
- Creación del archivo en `00-Inbox`
- Integración via MCP Obsidian

### **5. Registro en Historial**
- Grabación del envío exitoso
- Metadatos completos para tracking
- Estado y estadísticas de uso

## Configuración del Sistema

### **Archivo de Configuración**
```json
{
  "obsidian": {
    "enabled": true,
    "vault_path": "/Users/username/Documents/ObsidianVault",
    "ingest_folder": "00-Inbox",
    "default_template": "email_standard",
    "auto_organize": false, // Siempre va a 00-Inbox
    "filename_format": "{{date}}_{{subject_slug}}_{{from_domain}}",
    "templates_source": "obsidian", // "obsidian" o "local"
    "templates_path": "Templates/email_templates",
    "history_enabled": true,
    "prevent_duplicates": true,
    "max_file_size": 1048576, // 1MB
    "include_attachments": false
  }
}
```

### **Variables de Entorno**
```bash
# Opcional: Configuración via variables de entorno
OBSIDIAN_VAULT_PATH="/path/to/vault"
OBSIDIAN_INGEST_FOLDER="00-Inbox"
OBSIDIAN_DEFAULT_TEMPLATE="email_standard"
```

## Integración con MCP Obsidian

### **Uso del Servicio MCP Existente**
```go
// internal/services/obsidian_service.go
func (s *ObsidianServiceImpl) createObsidianNote(ctx context.Context, path, content string) error {
    // Usar el servicio MCP Obsidian existente
    // mcp_obsidian-mcp_read_notes y mcp_obsidian-mcp_search_notes
    
    // Crear la nota en Obsidian
    // Implementar según la API disponible del MCP
    return nil
}
```

### **Funcionalidades MCP Requeridas**
- **Creación de notas**: Crear archivos Markdown en Obsidian
- **Lectura de templates**: Leer plantillas existentes
- **Validación de conexión**: Verificar que Obsidian esté disponible

## Interfaz de Usuario

### **1. Modal de Ingesta**
- **Preview del contenido**: Mostrar cómo se verá el email en Obsidian
- **Selección de template**: Dropdown con templates disponibles
- **Confirmación de ruta**: Mostrar que irá a `00-Inbox`
- **Opciones adicionales**: Metadatos personalizados

### **2. Feedback del Usuario**
- **Progress**: Indicador de progreso durante la ingesta
- **Success**: Confirmación de ingesta exitosa
- **Error**: Manejo de errores con sugerencias de solución

### **3. Historial de Ingestas**
- **Vista de historial**: Lista de emails ingeridos recientemente
- **Búsqueda**: Filtrar por fecha, template, estado
- **Reingesta**: Opción para reingerir si es necesario

## Implementación por Fases

### **Fase 1: Funcionalidad Básica (Sprint 1)**
- [ ] Crear ObsidianService básico
- [ ] Implementar formateo Markdown simple
- [ ] Modal de ingesta básico
- [ ] Integración con tecla Shift+O
- [ ] Tabla de historial en base de datos

### **Fase 2: Templates y Organización (Sprint 2)**
- [ ] Sistema de templates en Obsidian
- [ ] Templates locales como fallback
- [ ] Organización automática en 00-Inbox
- [ ] Configuración avanzada
- [ ] Preview del contenido formateado

### **Fase 3: Integración MCP Completa (Sprint 3)**
- [ ] Uso directo del servicio MCP Obsidian
- [ ] Creación de notas en Obsidian
- [ ] Lectura de templates desde Obsidian
- [ ] Validación de conexión
- [ ] Manejo de errores robusto

### **Fase 4: Funcionalidades Avanzadas (Sprint 4)**
- [ ] Bulk ingesta de múltiples emails
- [ ] Templates personalizables por usuario
- [ ] Estadísticas de uso
- [ ] Exportación de historial
- [ ] Tests completos del sistema

## Consideraciones Técnicas

### **1. Thread Safety**
- Todas las operaciones de archivo en goroutines
- Uso de `QueueUpdateDraw` para actualizaciones de UI
- Accessors thread-safe para estado de la aplicación

### **2. Performance**
- Procesamiento asíncrono para emails grandes
- Cache de templates para evitar lecturas repetidas
- Límites de tamaño de archivo configurables

### **3. Error Handling**
- Uso del ErrorHandler centralizado
- Logging detallado para debugging
- Fallbacks para diferentes escenarios de error

### **4. Seguridad**
- Validación de rutas de archivo
- Sanitización de nombres de archivo
- Límites de tamaño y contenido

## Testing y Calidad

### **1. Tests Unitarios**
- **Servicios**: ObsidianService, ObsidianHistoryService
- **Formateo**: Templates y sustitución de variables
- **Validación**: Entrada de datos y configuración

### **2. Tests de Integración**
- **MCP Obsidian**: Integración con el servicio externo
- **Base de Datos**: Operaciones de historial
- **UI**: Flujo completo de ingesta

### **3. Tests de Performance**
- **Emails grandes**: Procesamiento de contenido extenso
- **Bulk ingesta**: Múltiples emails simultáneos
- **Templates complejos**: Sustitución de muchas variables

## Métricas y Monitoreo

### **1. Métricas de Uso**
- **Volumen**: Número de emails ingeridos por día/semana
- **Templates**: Uso de diferentes tipos de template
- **Errores**: Tasa de fallos y tipos de error

### **2. Performance**
- **Tiempo de ingesta**: Latencia promedio por email
- **Tamaño de archivo**: Distribución de tamaños
- **Uso de memoria**: Consumo durante operaciones

## Riesgos y Mitigaciones

### **1. Riesgos Identificados**
- **Disponibilidad de Obsidian**: Vault no accesible
- **Permisos de archivo**: Sin permisos de escritura
- **Tamaño de archivo**: Emails muy grandes
- **Conectividad MCP**: Servicio MCP no disponible

### **2. Estrategias de Mitigación**
- **Fallbacks**: Templates locales cuando Obsidian no esté disponible
- **Validación**: Verificación de permisos antes de la ingesta
- **Límites**: Tamaños máximos configurables
- **Retry logic**: Reintentos automáticos para fallos temporales

## Criterios de Aceptación

### **1. Funcionalidad Básica**
- [ ] Usuario puede ingerir email con Shift+O
- [ ] Email se guarda en 00-Inbox de Obsidian
- [ ] Formato Markdown correcto con metadatos
- [ ] Prevención de duplicados funciona

### **2. Templates y Personalización**
- [ ] Sistema de templates funcional
- [ ] Variables se sustituyen correctamente
- [ ] Templates se pueden editar en Obsidian
- [ ] Fallback a templates locales funciona

### **3. Historial y Tracking**
- [ ] Todos los envíos se registran
- [ ] Búsqueda en historial funciona
- [ ] Metadatos se almacenan correctamente
- [ ] Estadísticas de uso son precisas

### **4. Integración y UX**
- [ ] UI es intuitiva y responsiva
- [ ] Feedback del usuario es claro
- [ ] Manejo de errores es robusto
- [ ] Performance es aceptable

## Conclusión

Este sistema de ingesta a Obsidian representa una mejora significativa en el flujo de trabajo de Gmail TUI, permitiendo a los usuarios integrar sus emails con su sistema de segundo cerebro de manera eficiente y organizada. La arquitectura propuesta sigue los principios establecidos del proyecto, manteniendo la separación de responsabilidades y la modularidad del código.

La implementación por fases permite un desarrollo incremental y la validación temprana de la funcionalidad, mientras que las consideraciones técnicas aseguran un sistema robusto y mantenible a largo plazo.

---

**Documento creado**: {{date}}
**Versión**: 1.0
**Estado**: Propuesta
**Próximo paso**: Revisión y aprobación del equipo
