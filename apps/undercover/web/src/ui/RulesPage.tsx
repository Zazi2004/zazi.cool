import { Link } from "./router";

export function RulesPage() {
  return (
    <article class="stack prose">
      <h1>Règles du jeu</h1>

      <h2>Les rôles</h2>
      <ul>
        <li>
          <strong>Civils</strong> : ils reçoivent tous le même mot. Ils sont la majorité.
        </li>
        <li>
          <strong>Undercovers</strong> : ils reçoivent un mot proche, mais différent. Ils ne savent
          pas qu'ils sont undercover !
        </li>
        <li>
          <strong>Mr. White</strong> : il ne reçoit aucun mot, et il le sait. Il doit deviner de
          quoi parlent les autres.
        </li>
      </ul>

      <h2>Déroulé</h2>
      <ol>
        <li>Le téléphone passe de main en main : chacun découvre son mot en secret.</li>
        <li>
          À chaque tour, l'application tire un ordre de passage. Chacun donne, à l'oral, un indice
          sur son mot, sans le dire.
        </li>
        <li>
          Tout le monde vote à l'oral pour éliminer un joueur. Les rôles ne sont révélés qu'à la fin
          de la partie.
        </li>
        <li>
          Un Mr. White éliminé peut tenter de deviner le mot des civils. S'il trouve, il gagne seul.
        </li>
      </ol>

      <h2>Fin de partie</h2>
      <ul>
        <li>Les civils gagnent quand tous les undercovers et Mr. White sont éliminés.</li>
        <li>Les imposteurs gagnent quand il ne reste plus qu'un seul civil.</li>
        <li>Le premier Mr. White qui devine le mot des civils gagne seul.</li>
      </ul>

      <Link href="/partie" class="button primary">
        Lancer une partie
      </Link>
    </article>
  );
}
