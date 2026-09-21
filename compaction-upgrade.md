Voici le plan mis à jour en Markdown, simplifié pour se concentrer **uniquement sur l'intégration de JEV** (en supprimant la complexité d'un provider local). Tu peux le donner directement à ton agent pour qu'il mette à jour ton harnais.

---

# Plan d'implémentation : Système de Compaction & Routage (JEV Only)

## Objectif

Implémenter un système de compaction de contexte multiniveau pour notre harnais d'agent, combinant un nettoyage automatique et continu des résultats d'outils, un filtrage sémantique granulaire piloté exclusivement par l'API **JEV**, et une échelle de seuils par paliers.

---

## Tâches pour l'agent

### Tâche 1 : Définir le Contrat de Décision JEV

Créer les types stricts pour manipuler les retours de JEV dans le harnais.

* **Fichier cible :** `src/context/types.ts`
* **Actions :**
* Définir les types d'actions possibles : `export type ContextAction = 'conserver' | 'supprimer' | 'résumer';`
* Créer l'interface standardisée pour le résultat de JEV :
```typescript
export interface JevDecisionResult {
  action: ContextAction;
  confidence: number; // Probabilité calibrée entre 0 et 1 renvoyée par Jev
  summary?: string;   // Utilisé si l'action est 'résumer'
}

```





### Tâche 2 : Implémenter le Client JEV

Créer le service dédié aux appels de l'API JEV.

* **Fichier cible :** `src/context/jevClient.ts`
* **Actions :**
* Écrire une classe ou un ensemble de fonctions pour appeler l'API JEV.
* Formater les requêtes en envoyant l'état non structuré du bloc de texte et les choix prédéfinis (`conserver`, `supprimer`, `résumer`).
* Transformer les probabilités reçues de JEV en un objet `JevDecisionResult` exploitable.



### Tâche 3 : Implémenter le Nettoyage Continu des Outils (*Tool-Result Clearing*)

Mettre en place la purge automatique et immédiate des artéfacts techniques lourds à chaque tour pour empêcher l'explosion du contexte.

* **Fichier cible :** `src/context/pruning.ts`
* **Actions :**
* Écrire une fonction `clearOldToolResults(messages, options)` exécutée en continu.
* Conserver uniquement le dernier résultat d'outil brut nécessaire (`keepLastN: 1`), et remplacer les anciens par un placeholder léger (`[Tool output cleared to save context space]`).



### Tâche 4 : Implémenter l'Échelle de Seuils par Paliers (The Threshold Ladder)

Structurer l'orchestrateur global de compaction basé sur le ratio d'utilisation de la fenêtre de contexte.

* **Fichier cible :** `src/context/orchestrator.ts`
* **Actions :**
* Définir les seuils critiques :
* `WATCH`: 70% (Informatif)
* `PRUNE`: 85% (Déclenche l'appel à JEV pour analyser et filtrer granulairement les blocs textuels)
* `EMERGENCY`: 95% (Déclenche un résumé global d'urgence de l'historique ancien en préservant une *sliding window* de fin)


* Créer la fonction principale `manageContextCompaction(state)` qui orchestre :
1. Le nettoyage automatique des tools (en continu).
2. L'évaluation sémantique via JEV si le ratio dépasse 85%.





### Tâche 5 : Intégration dans la Boucle d'Exécution de l'Agent

Brancher l'orchestrateur de compaction dans la boucle de vie principale de l'agent.

* **Fichier cible :** `src/agent/harness.ts`
* **Actions :**
* Appeler l'orchestrateur de compaction à chaque fin de tour d'interaction.
* Réinjecter l'état du contexte nettoyé et optimisé dans le payload envoyé au modèle principal.
