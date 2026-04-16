# Doc TODOs

## DETAIL_DOCS.md
Erstelle das Dokument "DETAIL_DOCS.md" für Projekt "mlcartifact".

Vorgehen:
1. get_project_context für "mlcartifact" aufrufen
2. get_template für "DETAIL_DOCS.md" aufrufen
3. Lies das Projektverzeichnis gründlich um die nötigen Informationen zu sammeln:
   - Code: go.mod / package.json / Cargo.toml
   - Deployment: Dockerfile, docker-compose.yml, Taskfile.yml, *.service
   - Docs: README.md, INSTALL.md
4. Fülle das Template mit den gefundenen Informationen
   WICHTIG:
   - Keine Informationen duplizieren die bereits in anderen .mlcai/-Docs stehen.
     Stattdessen darauf verweisen (z.B. "siehe TECH_STACK.md für Details zum Stack").
   - Bei DETAIL_DOCS.md und USER_DOCS.md: NUR Dokumente ausserhalb von .mlcai/ auflisten
     (z.B. docs/, README.md, INSTALL.md). Die .mlcai/-Docs sind bereits über den Doc Hub
     erreichbar und gehören NICHT in diese Übersicht.
5. Zeige mir eine Zusammenfassung der Kernfakten zur Bestätigung
6. Nach Bestätigung: create_doc aufrufen (project_id: "mlcartifact",
   doc_type: "DETAIL_DOCS.md", author=DEIN_MODELL_IDENTIFIER)
7. update_worklog mit Zusammenfassung
Bereits vorhandene .mlcai/-Docs (nicht duplizieren — darauf verweisen wenn relevant):
  - INTEGRATION.md
  - TECH_STACK.md
  - WORKLOG.md

## API_CONTRACT.md
Erstelle das Dokument "API_CONTRACT.md" für Projekt "mlcartifact".

Vorgehen:
1. get_project_context für "mlcartifact" aufrufen
2. get_template für "API_CONTRACT.md" aufrufen
3. Lies das Projektverzeichnis gründlich um die nötigen Informationen zu sammeln:
   - Code: go.mod / package.json / Cargo.toml
   - Deployment: Dockerfile, docker-compose.yml, Taskfile.yml, *.service
   - Docs: README.md, INSTALL.md
4. Fülle das Template mit den gefundenen Informationen
   WICHTIG:
   - Keine Informationen duplizieren die bereits in anderen .mlcai/-Docs stehen.
     Stattdessen darauf verweisen (z.B. "siehe TECH_STACK.md für Details zum Stack").
   - Bei DETAIL_DOCS.md und USER_DOCS.md: NUR Dokumente ausserhalb von .mlcai/ auflisten
     (z.B. docs/, README.md, INSTALL.md). Die .mlcai/-Docs sind bereits über den Doc Hub
     erreichbar und gehören NICHT in diese Übersicht.
5. Zeige mir eine Zusammenfassung der Kernfakten zur Bestätigung
6. Nach Bestätigung: create_doc aufrufen (project_id: "mlcartifact",
   doc_type: "API_CONTRACT.md", author=DEIN_MODELL_IDENTIFIER)
7. update_worklog mit Zusammenfassung
Bereits vorhandene .mlcai/-Docs (nicht duplizieren — darauf verweisen wenn relevant):
  - INTEGRATION.md
  - TECH_STACK.md
  - WORKLOG.md

## DECISION_LOG.md
Erstelle das Dokument "DECISION_LOG.md" für Projekt "mlcartifact".

Vorgehen:
1. get_project_context für "mlcartifact" aufrufen
2. get_template für "DECISION_LOG.md" aufrufen
3. Lies das Projektverzeichnis gründlich um die nötigen Informationen zu sammeln:
   - Code: go.mod / package.json / Cargo.toml
   - Deployment: Dockerfile, docker-compose.yml, Taskfile.yml, *.service
   - Docs: README.md, INSTALL.md
4. Fülle das Template mit den gefundenen Informationen
   WICHTIG:
   - Keine Informationen duplizieren die bereits in anderen .mlcai/-Docs stehen.
     Stattdessen darauf verweisen (z.B. "siehe TECH_STACK.md für Details zum Stack").
   - Bei DETAIL_DOCS.md und USER_DOCS.md: NUR Dokumente ausserhalb von .mlcai/ auflisten
     (z.B. docs/, README.md, INSTALL.md). Die .mlcai/-Docs sind bereits über den Doc Hub
     erreichbar und gehören NICHT in diese Übersicht.
5. Zeige mir eine Zusammenfassung der Kernfakten zur Bestätigung
6. Nach Bestätigung: create_doc aufrufen (project_id: "mlcartifact",
   doc_type: "DECISION_LOG.md", author=DEIN_MODELL_IDENTIFIER)
7. update_worklog mit Zusammenfassung
Bereits vorhandene .mlcai/-Docs (nicht duplizieren — darauf verweisen wenn relevant):
  - INTEGRATION.md
  - TECH_STACK.md
  - WORKLOG.md

## USER_DOCS.md
Erstelle das Dokument "USER_DOCS.md" für Projekt "mlcartifact".

Vorgehen:
1. get_project_context für "mlcartifact" aufrufen
2. get_template für "USER_DOCS.md" aufrufen
3. Lies das Projektverzeichnis gründlich um die nötigen Informationen zu sammeln:
   - Code: go.mod / package.json / Cargo.toml
   - Deployment: Dockerfile, docker-compose.yml, Taskfile.yml, *.service
   - Docs: README.md, INSTALL.md
4. Fülle das Template mit den gefundenen Informationen
   WICHTIG:
   - Keine Informationen duplizieren die bereits in anderen .mlcai/-Docs stehen.
     Stattdessen darauf verweisen (z.B. "siehe TECH_STACK.md für Details zum Stack").
   - Bei DETAIL_DOCS.md und USER_DOCS.md: NUR Dokumente ausserhalb von .mlcai/ auflisten
     (z.B. docs/, README.md, INSTALL.md). Die .mlcai/-Docs sind bereits über den Doc Hub
     erreichbar und gehören NICHT in diese Übersicht.
5. Zeige mir eine Zusammenfassung der Kernfakten zur Bestätigung
6. Nach Bestätigung: create_doc aufrufen (project_id: "mlcartifact",
   doc_type: "USER_DOCS.md", author=DEIN_MODELL_IDENTIFIER)
7. update_worklog mit Zusammenfassung
Bereits vorhandene .mlcai/-Docs (nicht duplizieren — darauf verweisen wenn relevant):
  - INTEGRATION.md
  - TECH_STACK.md
  - WORKLOG.md

## TECH_STACK.md
Prüfe und aktualisiere das Dokument "TECH_STACK.md" für Projekt "mlcartifact".

Vorgehen:
1. get_project_context für "mlcartifact" aufrufen
2. get_doc aufrufen (project_id: "mlcartifact", doc_type: "TECH_STACK.md")
3. Lies den aktuellen Inhalt und prüfe:
   - Sind alle Informationen noch aktuell? (Pfade, Ports, Versionen, Dependencies)
   - Fehlen wichtige Abschnitte?
   - Stimmt der Inhalt mit dem tatsächlichen Code überein?
4. Falls Änderungen nötig: update_section oder create_doc mit base_modified
   und author=DEIN_MODELL_IDENTIFIER
5. update_worklog mit Zusammenfassung
Bereits vorhandene .mlcai/-Docs (nicht duplizieren — darauf verweisen wenn relevant):
  - INTEGRATION.md
  - TECH_STACK.md
  - WORKLOG.md

## INTEGRATION.md
Prüfe und aktualisiere das Dokument "INTEGRATION.md" für Projekt "mlcartifact".

Vorgehen:
1. get_project_context für "mlcartifact" aufrufen
2. get_doc aufrufen (project_id: "mlcartifact", doc_type: "INTEGRATION.md")
3. Lies den aktuellen Inhalt und prüfe:
   - Sind alle Informationen noch aktuell? (Pfade, Ports, Versionen, Dependencies)
   - Fehlen wichtige Abschnitte?
   - Stimmt der Inhalt mit dem tatsächlichen Code überein?
4. Falls Änderungen nötig: update_section oder create_doc mit base_modified
   und author=DEIN_MODELL_IDENTIFIER
5. update_worklog mit Zusammenfassung
Bereits vorhandene .mlcai/-Docs (nicht duplizieren — darauf verweisen wenn relevant):
  - INTEGRATION.md
  - TECH_STACK.md
  - WORKLOG.md
