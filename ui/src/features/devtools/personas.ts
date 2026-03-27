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
    name: "Emily",
    subtitle: "Ovulatory, Regular",
    fixturePath: "/fixtures/regular-12.json",
  },
  {
    id: "ovulatory-somewhat-irregular",
    name: "Jessica",
    subtitle: "Ovulatory, Somewhat Irregular",
    fixturePath: "/fixtures/ovulatory-somewhat-irregular.json",
  },
  {
    id: "ovulatory-very-irregular",
    name: "Ashley",
    subtitle: "Ovulatory, Very Irregular",
    fixturePath: "/fixtures/ovulatory-very-irregular.json",
  },
  {
    id: "ovulatory-unknown",
    name: "Sophie",
    subtitle: "Ovulatory, Unknown Regularity",
    fixturePath: "/fixtures/ovulatory-unknown.json",
  },
  {
    id: "hormonal-regular",
    name: "Laura",
    subtitle: "Hormonally Suppressed, Regular",
    fixturePath: "/fixtures/hormonal-regular.json",
  },
  {
    id: "hormonal-somewhat-irregular",
    name: "Emma",
    subtitle: "Hormonally Suppressed, Somewhat Irregular",
    fixturePath: "/fixtures/hormonal-somewhat-irregular.json",
  },
  {
    id: "hormonal-very-irregular",
    name: "Camille",
    subtitle: "Hormonally Suppressed, Very Irregular",
    fixturePath: "/fixtures/hormonal-very-irregular.json",
  },
  {
    id: "irregular",
    name: "Hannah",
    subtitle: "Irregular Cycle Model",
    fixturePath: "/fixtures/irregular.json",
  },
  {
    id: "irregular-very-irregular",
    name: "Priya",
    subtitle: "Irregular, Very Irregular",
    fixturePath: "/fixtures/irregular-very-irregular.json",
  },
];
