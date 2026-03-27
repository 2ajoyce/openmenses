export interface Persona {
  /** Unique scenario key matching the fixture filename. */
  id: string;
  /** Human name for display. */
  name: string;
  /** Descriptive subtitle (model + regularity). */
  subtitle: string;
  /** Path to the pre-generated fixture JSON file (relative to public/). */
  fixturePath: string;
}

export const personas: Persona[] = [
  {
    id: "regular-12",
    name: "Amara",
    subtitle: "Ovulatory, Regular",
    fixturePath: "/fixtures/regular-12.json",
  },
  {
    id: "ovulatory-somewhat-irregular",
    name: "Beatriz",
    subtitle: "Ovulatory, Somewhat Irregular",
    fixturePath: "/fixtures/ovulatory-somewhat-irregular.json",
  },
  {
    id: "ovulatory-very-irregular",
    name: "Chioma",
    subtitle: "Ovulatory, Very Irregular",
    fixturePath: "/fixtures/ovulatory-very-irregular.json",
  },
  {
    id: "ovulatory-unknown",
    name: "Diya",
    subtitle: "Ovulatory, Unknown Regularity",
    fixturePath: "/fixtures/ovulatory-unknown.json",
  },
  {
    id: "hormonal-regular",
    name: "Elena",
    subtitle: "Hormonally Suppressed, Regular",
    fixturePath: "/fixtures/hormonal-regular.json",
  },
  {
    id: "hormonal-somewhat-irregular",
    name: "Fatou",
    subtitle: "Hormonally Suppressed, Somewhat Irregular",
    fixturePath: "/fixtures/hormonal-somewhat-irregular.json",
  },
  {
    id: "hormonal-very-irregular",
    name: "Greta",
    subtitle: "Hormonally Suppressed, Very Irregular",
    fixturePath: "/fixtures/hormonal-very-irregular.json",
  },
  {
    id: "irregular",
    name: "Hana",
    subtitle: "Irregular Cycle Model",
    fixturePath: "/fixtures/irregular.json",
  },
  {
    id: "irregular-very-irregular",
    name: "Ingrid",
    subtitle: "Irregular, Very Irregular",
    fixturePath: "/fixtures/irregular-very-irregular.json",
  },
];
